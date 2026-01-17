package rest

import (
	"encoding/json"
	"net/http"

	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// SObjectResponse is the response for create/update operations
type SObjectResponse struct {
	ID      string        `json:"id"`
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

// handleDescribeGlobal handles GET /services/data/vXX.X/sobjects/
func (r *Router) handleDescribeGlobal(w http.ResponseWriter, req *http.Request, params []string) {
	description, err := r.store.DescribeGlobal()
	if err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusInternalServerError)
		return
	}

	r.respondJSON(w, description, http.StatusOK)
}

// handleSObject handles GET/POST /services/data/vXX.X/sobjects/{objectType}/
func (r *Router) handleSObject(w http.ResponseWriter, req *http.Request, params []string) {
	objectType := params[0]

	switch req.Method {
	case "GET":
		r.handleDescribeSObject(w, req, params)
	case "POST":
		r.handleCreateRecord(w, req, objectType)
	}
}

// handleDescribeSObject handles GET /services/data/vXX.X/sobjects/{objectType}/describe
func (r *Router) handleDescribeSObject(w http.ResponseWriter, req *http.Request, params []string) {
	objectType := params[0]

	description, err := r.store.DescribeSObject(objectType)
	if err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewObjectNotFoundError(objectType),
		}, http.StatusNotFound)
		return
	}

	r.respondJSON(w, description, http.StatusOK)
}

// handleCreateRecord handles POST /services/data/vXX.X/sobjects/{objectType}/
func (r *Router) handleCreateRecord(w http.ResponseWriter, req *http.Request, objectType string) {
	// Check if object type exists
	if !r.store.HasSObject(objectType) {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewInvalidTypeError(objectType),
		}, http.StatusNotFound)
		return
	}

	// Parse request body
	var record storage.Record
	if err := json.NewDecoder(req.Body).Decode(&record); err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	// Create record
	id, err := r.store.CreateRecord(objectType, record)
	if err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	response := SObjectResponse{
		ID:      id,
		Success: true,
		Errors:  []interface{}{},
	}

	r.respondJSON(w, response, http.StatusCreated)
}

// handleSObjectRecord handles GET/PATCH/DELETE /services/data/vXX.X/sobjects/{objectType}/{recordID}
func (r *Router) handleSObjectRecord(w http.ResponseWriter, req *http.Request, params []string) {
	objectType := params[0]
	recordID := params[1]

	// Check if object type exists
	if !r.store.HasSObject(objectType) {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewInvalidTypeError(objectType),
		}, http.StatusNotFound)
		return
	}

	switch req.Method {
	case "GET":
		r.handleGetRecord(w, req, objectType, recordID)
	case "PATCH":
		r.handleUpdateRecord(w, req, objectType, recordID)
	case "DELETE":
		r.handleDeleteRecord(w, req, objectType, recordID)
	}
}

// handleGetRecord handles GET /services/data/vXX.X/sobjects/{objectType}/{recordID}
func (r *Router) handleGetRecord(w http.ResponseWriter, req *http.Request, objectType, recordID string) {
	record, err := r.store.GetRecord(objectType, recordID)
	if err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewNotFoundError(objectType, recordID),
		}, http.StatusNotFound)
		return
	}

	// Handle field selection
	fields := req.URL.Query().Get("fields")
	if fields != "" {
		record = selectFields(record, fields)
	}

	r.respondJSON(w, record, http.StatusOK)
}

// handleUpdateRecord handles PATCH /services/data/vXX.X/sobjects/{objectType}/{recordID}
func (r *Router) handleUpdateRecord(w http.ResponseWriter, req *http.Request, objectType, recordID string) {
	// Parse request body
	var updates storage.Record
	if err := json.NewDecoder(req.Body).Decode(&updates); err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	// Update record
	err := r.store.UpdateRecord(objectType, recordID, updates)
	if err != nil {
		if err.Error() == "record not found: "+recordID {
			r.respondError(w, []sferrors.SalesforceError{
				sferrors.NewNotFoundError(objectType, recordID),
			}, http.StatusNotFound)
			return
		}
		r.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	// 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}

// handleDeleteRecord handles DELETE /services/data/vXX.X/sobjects/{objectType}/{recordID}
func (r *Router) handleDeleteRecord(w http.ResponseWriter, req *http.Request, objectType, recordID string) {
	err := r.store.DeleteRecord(objectType, recordID)
	if err != nil {
		if err.Error() == "record not found: "+recordID {
			r.respondError(w, []sferrors.SalesforceError{
				sferrors.NewNotFoundError(objectType, recordID),
			}, http.StatusNotFound)
			return
		}
		r.respondError(w, []sferrors.SalesforceError{
			{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	// 204 No Content on success
	w.WriteHeader(http.StatusNoContent)
}

// selectFields filters a record to only include specified fields
func selectFields(record storage.Record, fieldsStr string) storage.Record {
	if fieldsStr == "" {
		return record
	}

	fieldList := parseFieldList(fieldsStr)
	result := make(storage.Record)

	// Always include Id and attributes
	if id, ok := record["Id"]; ok {
		result["Id"] = id
	}
	if attrs, ok := record["attributes"]; ok {
		result["attributes"] = attrs
	}

	for _, field := range fieldList {
		if val, ok := record[field]; ok {
			result[field] = val
		}
	}

	return result
}

// parseFieldList parses a comma-separated field list
func parseFieldList(fields string) []string {
	var result []string
	for _, f := range splitAndTrim(fields, ",") {
		if f != "" {
			result = append(result, f)
		}
	}
	return result
}

// splitAndTrim splits a string and trims each element
func splitAndTrim(s, sep string) []string {
	parts := make([]string, 0)
	for _, p := range stringsSplit(s, sep) {
		trimmed := stringsTrim(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

func stringsSplit(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
			i += len(sep) - 1
		}
	}
	result = append(result, s[start:])
	return result
}

func stringsTrim(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}
