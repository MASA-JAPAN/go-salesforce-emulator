package bulk

import (
	"encoding/csv"
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// Handler handles Bulk API requests
type Handler struct {
	store       storage.Store
	authHandler *auth.Handler
	apiVersion  string
}

// NewHandler creates a new bulk API handler
func NewHandler(store storage.Store, authHandler *auth.Handler, apiVersion string) *Handler {
	return &Handler{
		store:       store,
		authHandler: authHandler,
		apiVersion:  apiVersion,
	}
}

// JobRequest represents a request to create a bulk job
type JobRequest struct {
	Operation   string `json:"operation"`
	Query       string `json:"query"`
	ContentType string `json:"contentType,omitempty"`
}

// JobResponse represents a bulk job response
type JobResponse struct {
	ID                     string  `json:"id"`
	Operation              string  `json:"operation"`
	Object                 string  `json:"object"`
	CreatedById            string  `json:"createdById"`
	CreatedDate            string  `json:"createdDate"`
	SystemModstamp         string  `json:"systemModstamp"`
	State                  string  `json:"state"`
	ConcurrencyMode        string  `json:"concurrencyMode"`
	ContentType            string  `json:"contentType"`
	ApiVersion             float64 `json:"apiVersion"`
	JobType                string  `json:"jobType"`
	LineEnding             string  `json:"lineEnding"`
	ColumnDelimiter        string  `json:"columnDelimiter"`
	NumberRecordsProcessed int     `json:"numberRecordsProcessed"`
	Retries                int     `json:"retries"`
	TotalProcessingTime    int     `json:"totalProcessingTime"`
}

// HandleJobs handles POST/GET /services/data/vXX.X/jobs/query
func (h *Handler) HandleJobs(w http.ResponseWriter, r *http.Request) {
	// Validate auth
	if _, err := h.authHandler.ValidateRequest(r); err != nil {
		h.respondError(w, []sferrors.SalesforceError{err.(sferrors.SalesforceError)}, http.StatusUnauthorized)
		return
	}

	switch r.Method {
	case "POST":
		h.handleCreateJob(w, r)
	case "GET":
		h.handleListJobs(w, r)
	default:
		h.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMethodNotAllowedError(r.Method),
		}, http.StatusMethodNotAllowed)
	}
}

// HandleJobByID handles requests to /services/data/vXX.X/jobs/query/{jobId}[/results]
func (h *Handler) HandleJobByID(w http.ResponseWriter, r *http.Request) {
	// Validate auth
	if _, err := h.authHandler.ValidateRequest(r); err != nil {
		h.respondError(w, []sferrors.SalesforceError{err.(sferrors.SalesforceError)}, http.StatusUnauthorized)
		return
	}

	// Parse job ID from path
	path := r.URL.Path
	pattern := regexp.MustCompile(`/jobs/query/([^/]+)(/results)?`)
	matches := pattern.FindStringSubmatch(path)

	if len(matches) < 2 {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Invalid job ID", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	jobID := matches[1]
	isResults := len(matches) > 2 && matches[2] == "/results"

	if isResults {
		h.handleGetResults(w, r, jobID)
		return
	}

	switch r.Method {
	case "GET":
		h.handleGetJob(w, r, jobID)
	case "PATCH":
		h.handleUpdateJob(w, r, jobID)
	case "DELETE":
		h.handleDeleteJob(w, r, jobID)
	default:
		h.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMethodNotAllowedError(r.Method),
		}, http.StatusMethodNotAllowed)
	}
}

// handleCreateJob handles POST /services/data/vXX.X/jobs/query
func (h *Handler) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req JobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	// Extract object type from query
	objectType := extractObjectFromQuery(req.Query)
	if objectType == "" {
		h.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMalformedQueryError("Unable to parse object from query"),
		}, http.StatusBadRequest)
		return
	}

	// Create bulk job
	job, err := h.store.CreateBulkJob(storage.BulkJobConfig{
		Operation:   req.Operation,
		Object:      objectType,
		Query:       req.Query,
		ContentType: req.ContentType,
	})
	if err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	// Take a snapshot of the job for the response before starting async processing
	// to avoid data race between the goroutine modifying job state and response serialization
	response := h.jobToResponse(job)

	// Process the job immediately (in a real implementation, this would be async)
	go h.processJob(job.ID, req.Query)

	h.respondJSON(w, response, http.StatusOK)
}

// handleListJobs handles GET /services/data/vXX.X/jobs/query
func (h *Handler) handleListJobs(w http.ResponseWriter, r *http.Request) {
	// For simplicity, return an empty list
	// In a real implementation, we'd iterate through all jobs
	response := map[string]interface{}{
		"done":    true,
		"records": []interface{}{},
	}
	h.respondJSON(w, response, http.StatusOK)
}

// handleGetJob handles GET /services/data/vXX.X/jobs/query/{jobId}
func (h *Handler) handleGetJob(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := h.store.GetBulkJob(jobID)
	if err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Job not found", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	h.respondJSON(w, h.jobToResponse(job), http.StatusOK)
}

// handleUpdateJob handles PATCH /services/data/vXX.X/jobs/query/{jobId}
func (h *Handler) handleUpdateJob(w http.ResponseWriter, r *http.Request, jobID string) {
	var req struct {
		State string `json:"state"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	// Handle abort
	if req.State == "Aborted" {
		if err := h.store.UpdateBulkJobState(jobID, storage.JobStateAborted); err != nil {
			h.respondError(w, []sferrors.SalesforceError{
				{Message: err.Error(), ErrorCode: sferrors.ErrorCodeNotFound},
			}, http.StatusNotFound)
			return
		}
	}

	job, err := h.store.GetBulkJob(jobID)
	if err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Job not found", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	h.respondJSON(w, h.jobToResponse(job), http.StatusOK)
}

// handleDeleteJob handles DELETE /services/data/vXX.X/jobs/query/{jobId}
func (h *Handler) handleDeleteJob(w http.ResponseWriter, r *http.Request, jobID string) {
	if err := h.store.DeleteBulkJob(jobID); err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Job not found", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// handleGetResults handles GET /services/data/vXX.X/jobs/query/{jobId}/results
func (h *Handler) handleGetResults(w http.ResponseWriter, r *http.Request, jobID string) {
	job, err := h.store.GetBulkJob(jobID)
	if err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Job not found", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	if job.State != storage.JobStateJobComplete {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: "Job not complete", ErrorCode: sferrors.ErrorCodeInvalidOperation},
		}, http.StatusBadRequest)
		return
	}

	// Get pagination parameters
	locator := r.URL.Query().Get("locator")
	maxRecords := 2000
	if maxStr := r.URL.Query().Get("maxRecords"); maxStr != "" {
		if m, err := strconv.Atoi(maxStr); err == nil && m > 0 {
			maxRecords = m
		}
	}

	results, nextLocator, err := h.store.GetBulkJobResults(jobID, locator, maxRecords)
	if err != nil {
		h.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidOperation},
		}, http.StatusBadRequest)
		return
	}

	// Set Sforce-Locator header
	if nextLocator != "" {
		w.Header().Set("Sforce-Locator", nextLocator)
	} else {
		w.Header().Set("Sforce-Locator", "null")
	}

	// Write CSV response
	w.Header().Set("Content-Type", "text/csv")
	w.WriteHeader(http.StatusOK)

	h.writeCSV(w, results.Records)
}

// processJob processes a bulk query job
func (h *Handler) processJob(jobID, query string) {
	// Update state to InProgress
	h.store.UpdateBulkJobState(jobID, storage.JobStateInProgress)

	// Execute the query
	records, err := h.executeQuery(query)
	if err != nil {
		h.store.UpdateBulkJobState(jobID, storage.JobStateFailed)
		return
	}

	// Store results
	h.store.SetBulkJobResults(jobID, records)

	// Update state to Complete
	h.store.UpdateBulkJobState(jobID, storage.JobStateJobComplete)
}

// executeQuery executes a SOQL query for bulk processing
func (h *Handler) executeQuery(query string) ([]storage.Record, error) {
	// Parse FROM clause to get object type
	fromMatch := regexp.MustCompile(`(?i)FROM\s+(\w+)`).FindStringSubmatch(query)
	if fromMatch == nil {
		return nil, nil
	}
	objectType := fromMatch[1]

	// Get all records (simplified - doesn't handle full SOQL)
	records, err := h.store.GetAllRecords(objectType)
	if err != nil {
		return nil, err
	}

	// Parse SELECT fields
	selectMatch := regexp.MustCompile(`(?i)^SELECT\s+(.+?)\s+FROM\s+`).FindStringSubmatch(query)
	if selectMatch == nil {
		return records, nil
	}

	fields := parseFields(selectMatch[1])

	// Project fields
	result := make([]storage.Record, len(records))
	for i, record := range records {
		projected := make(storage.Record)
		for _, field := range fields {
			if val, ok := record[field]; ok {
				projected[field] = val
			}
		}
		result[i] = projected
	}

	return result, nil
}

// writeCSV writes records as CSV
func (h *Handler) writeCSV(w http.ResponseWriter, records []storage.Record) {
	writer := csv.NewWriter(w)
	defer writer.Flush()

	if len(records) == 0 {
		return
	}

	// Get headers from first record
	var headers []string
	for key := range records[0] {
		if key != "attributes" {
			headers = append(headers, key)
		}
	}

	// Write header row
	writer.Write(headers)

	// Write data rows
	for _, record := range records {
		row := make([]string, len(headers))
		for i, header := range headers {
			val := record[header]
			if val == nil {
				row[i] = ""
			} else {
				row[i] = toString(val)
			}
		}
		writer.Write(row)
	}
}

func (h *Handler) jobToResponse(job *storage.BulkJob) JobResponse {
	return JobResponse{
		ID:                     job.ID,
		Operation:              job.Operation,
		Object:                 job.Object,
		CreatedById:            job.CreatedById,
		CreatedDate:            job.CreatedDate.Format("2006-01-02T15:04:05.000+0000"),
		SystemModstamp:         job.SystemModstamp.Format("2006-01-02T15:04:05.000+0000"),
		State:                  string(job.State),
		ConcurrencyMode:        job.ConcurrencyMode,
		ContentType:            job.ContentType,
		ApiVersion:             job.ApiVersion,
		JobType:                job.JobType,
		LineEnding:             "LF",
		ColumnDelimiter:        "COMMA",
		NumberRecordsProcessed: job.NumberRecordsProcessed,
		Retries:                0,
		TotalProcessingTime:    0,
	}
}

func (h *Handler) respondError(w http.ResponseWriter, errors []sferrors.SalesforceError, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errors)
}

func (h *Handler) respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func extractObjectFromQuery(query string) string {
	match := regexp.MustCompile(`(?i)FROM\s+(\w+)`).FindStringSubmatch(query)
	if match != nil {
		return match[1]
	}
	return ""
}

func parseFields(fieldsStr string) []string {
	var fields []string
	for _, f := range strings.Split(fieldsStr, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

func toString(val interface{}) string {
	if val == nil {
		return ""
	}
	switch v := val.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}
