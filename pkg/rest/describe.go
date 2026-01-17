package rest

import (
	"net/http"

	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// handleLimits handles GET /services/data/vXX.X/limits
func (r *Router) handleLimits(w http.ResponseWriter, req *http.Request, params []string) {
	limits := r.store.GetLimits()
	r.respondJSON(w, limits, http.StatusOK)
}

// RecordCountResponse represents the response for record count API
type RecordCountResponse struct {
	SObjects []RecordCount `json:"sObjects"`
}

// RecordCount represents a single object's record count
type RecordCount struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// handleRecordCount handles GET /services/data/vXX.X/limits/recordCount
func (r *Router) handleRecordCount(w http.ResponseWriter, req *http.Request, params []string) {
	// Get the sObjects parameter
	sobjects := req.URL.Query().Get("sObjects")
	if sobjects == "" {
		r.respondError(w, []sferrors.SalesforceError{
			{Message: "sObjects parameter is required", ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	objectList := splitAndTrim(sobjects, ",")
	counts := r.store.GetRecordCounts(objectList)

	response := RecordCountResponse{
		SObjects: make([]RecordCount, 0, len(counts)),
	}

	for name, count := range counts {
		response.SObjects = append(response.SObjects, RecordCount{
			Name:  name,
			Count: count,
		})
	}

	r.respondJSON(w, response, http.StatusOK)
}

// handleToolingQuery handles GET /services/data/vXX.X/tooling/query
func (r *Router) handleToolingQuery(w http.ResponseWriter, req *http.Request, params []string) {
	query := req.URL.Query().Get("q")
	if query == "" {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMalformedQueryError("No query string provided"),
		}, http.StatusBadRequest)
		return
	}

	// For tooling API, we can reuse the SOQL parser but with tooling objects
	// For now, return an empty result set as tooling objects aren't fully implemented
	response := QueryResponse{
		TotalSize: 0,
		Done:      true,
		Records:   []storage.Record{},
	}

	r.respondJSON(w, response, http.StatusOK)
}

// handleToolingSObject handles GET/POST /services/data/vXX.X/tooling/sobjects/{type}
func (r *Router) handleToolingSObject(w http.ResponseWriter, req *http.Request, params []string) {
	objectType := params[0]

	switch req.Method {
	case "GET":
		// Return describe for tooling object
		r.respondJSON(w, map[string]interface{}{
			"name":       objectType,
			"label":      objectType,
			"queryable":  true,
			"createable": true,
			"updateable": true,
			"deletable":  true,
			"fields":     []interface{}{},
		}, http.StatusOK)
	case "POST":
		// Create tooling record (simplified - just return success)
		r.respondJSON(w, SObjectResponse{
			ID:      "00N000000000000AAA",
			Success: true,
			Errors:  []interface{}{},
		}, http.StatusCreated)
	}
}
