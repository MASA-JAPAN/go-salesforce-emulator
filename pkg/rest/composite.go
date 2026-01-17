package rest

import (
	"encoding/json"
	"net/http"
	"strings"

	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// CompositeRequest represents a composite API request
type CompositeRequest struct {
	AllOrNone        bool               `json:"allOrNone"`
	CompositeRequest []CompositeSubrequest `json:"compositeRequest"`
}

// CompositeSubrequest represents a single request in a composite batch
type CompositeSubrequest struct {
	Method      string                 `json:"method"`
	URL         string                 `json:"url"`
	ReferenceID string                 `json:"referenceId"`
	Body        map[string]interface{} `json:"body,omitempty"`
}

// CompositeResponse represents the response from a composite request
type CompositeResponse struct {
	CompositeResponse []CompositeSubresponse `json:"compositeResponse"`
}

// CompositeSubresponse represents a single response in a composite batch
type CompositeSubresponse struct {
	Body           interface{} `json:"body"`
	HTTPHeaders    map[string]string `json:"httpHeaders"`
	HTTPStatusCode int         `json:"httpStatusCode"`
	ReferenceID    string      `json:"referenceId"`
}

// CompositeSObjectsRequest represents a composite SObjects request
type CompositeSObjectsRequest struct {
	AllOrNone bool               `json:"allOrNone"`
	Records   []storage.Record   `json:"records"`
}

// handleCompositeSObjects handles POST/PATCH/DELETE /services/data/vXX.X/composite/sobjects
func (r *Router) handleCompositeSObjects(w http.ResponseWriter, req *http.Request, params []string) {
	switch req.Method {
	case "POST":
		r.handleCompositeCreate(w, req)
	case "PATCH":
		r.handleCompositeUpdate(w, req)
	case "DELETE":
		r.handleCompositeDelete(w, req)
	}
}

// handleCompositeCreate handles batch create operations
func (r *Router) handleCompositeCreate(w http.ResponseWriter, req *http.Request) {
	var request CompositeSObjectsRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	var results []SObjectResponse
	var hasError bool

	for _, record := range request.Records {
		// Get object type from attributes
		attrs, ok := record["attributes"].(map[string]interface{})
		if !ok {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{"Missing attributes.type"},
			})
			hasError = true
			continue
		}

		objectType, ok := attrs["type"].(string)
		if !ok {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{"Missing attributes.type"},
			})
			hasError = true
			continue
		}

		// Remove attributes from record before creating
		delete(record, "attributes")

		id, err := r.store.CreateRecord(objectType, record)
		if err != nil {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{err.Error()},
			})
			hasError = true
		} else {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			})
		}
	}

	// If allOrNone is true and there was an error, rollback would happen here
	// For simplicity, we're not implementing true transactions in the emulator

	if hasError && request.AllOrNone {
		r.respondJSON(w, results, http.StatusBadRequest)
		return
	}

	r.respondJSON(w, results, http.StatusCreated)
}

// handleCompositeUpdate handles batch update operations
func (r *Router) handleCompositeUpdate(w http.ResponseWriter, req *http.Request) {
	var request CompositeSObjectsRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	var results []SObjectResponse
	var hasError bool

	for _, record := range request.Records {
		// Get object type from attributes
		attrs, ok := record["attributes"].(map[string]interface{})
		if !ok {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{"Missing attributes.type"},
			})
			hasError = true
			continue
		}

		objectType, ok := attrs["type"].(string)
		if !ok {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{"Missing attributes.type"},
			})
			hasError = true
			continue
		}

		// Get record ID
		id, ok := record["Id"].(string)
		if !ok {
			results = append(results, SObjectResponse{
				Success: false,
				Errors:  []interface{}{"Missing Id field"},
			})
			hasError = true
			continue
		}

		// Remove attributes and Id from record before updating
		delete(record, "attributes")
		delete(record, "Id")

		err := r.store.UpdateRecord(objectType, id, record)
		if err != nil {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: false,
				Errors:  []interface{}{err.Error()},
			})
			hasError = true
		} else {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			})
		}
	}

	if hasError && request.AllOrNone {
		r.respondJSON(w, results, http.StatusBadRequest)
		return
	}

	// Return 204 No Content on successful update
	w.WriteHeader(http.StatusNoContent)
}

// handleCompositeDelete handles batch delete operations
func (r *Router) handleCompositeDelete(w http.ResponseWriter, req *http.Request) {
	// Get IDs from query parameter
	ids := req.URL.Query().Get("ids")
	if ids == "" {
		r.respondError(w, []sferrors.SalesforceError{
			{Message: "Missing ids parameter", ErrorCode: sferrors.ErrorCodeInvalidField},
		}, http.StatusBadRequest)
		return
	}

	allOrNone := req.URL.Query().Get("allOrNone") == "true"

	idList := strings.Split(ids, ",")
	var results []SObjectResponse
	var hasError bool

	for _, id := range idList {
		id = strings.TrimSpace(id)

		// Try to find and delete the record
		// We need to determine the object type from the ID prefix
		objectType := determineObjectType(id)
		if objectType == "" {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: false,
				Errors:  []interface{}{"Unable to determine object type from ID"},
			})
			hasError = true
			continue
		}

		err := r.store.DeleteRecord(objectType, id)
		if err != nil {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: false,
				Errors:  []interface{}{err.Error()},
			})
			hasError = true
		} else {
			results = append(results, SObjectResponse{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			})
		}
	}

	if hasError && allOrNone {
		r.respondJSON(w, results, http.StatusBadRequest)
		return
	}

	r.respondJSON(w, results, http.StatusOK)
}

// handleComposite handles POST /services/data/vXX.X/composite
func (r *Router) handleComposite(w http.ResponseWriter, req *http.Request, params []string) {
	var request CompositeRequest
	if err := json.NewDecoder(req.Body).Decode(&request); err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewJSONParserError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	response := CompositeResponse{
		CompositeResponse: make([]CompositeSubresponse, len(request.CompositeRequest)),
	}

	// Reference ID to result mapping for variable substitution
	refResults := make(map[string]interface{})

	for i, subreq := range request.CompositeRequest {
		// Substitute reference IDs in URL and body
		url := substituteReferences(subreq.URL, refResults)
		body := substituteBodyReferences(subreq.Body, refResults)

		subresponse := r.executeSubrequest(subreq.Method, url, body)
		subresponse.ReferenceID = subreq.ReferenceID

		response.CompositeResponse[i] = subresponse

		// Store result for reference substitution
		refResults[subreq.ReferenceID] = subresponse.Body
	}

	r.respondJSON(w, response, http.StatusOK)
}

// executeSubrequest executes a single composite subrequest
func (r *Router) executeSubrequest(method, url string, body map[string]interface{}) CompositeSubresponse {
	// This is a simplified implementation
	// In a real implementation, we would route this through the normal HTTP handler

	response := CompositeSubresponse{
		HTTPHeaders: map[string]string{
			"Content-Type": "application/json",
		},
	}

	// Parse the URL to determine the operation
	// Example: /services/data/v58.0/sobjects/Account/001...
	if strings.Contains(url, "/sobjects/") {
		parts := strings.Split(url, "/sobjects/")
		if len(parts) == 2 {
			pathParts := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
			objectType := pathParts[0]

			switch method {
			case "POST":
				id, err := r.store.CreateRecord(objectType, body)
				if err != nil {
					response.HTTPStatusCode = 400
					response.Body = []sferrors.SalesforceError{{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField}}
				} else {
					response.HTTPStatusCode = 201
					response.Body = SObjectResponse{ID: id, Success: true, Errors: []interface{}{}}
				}

			case "GET":
				if len(pathParts) > 1 {
					recordID := pathParts[1]
					record, err := r.store.GetRecord(objectType, recordID)
					if err != nil {
						response.HTTPStatusCode = 404
						response.Body = []sferrors.SalesforceError{sferrors.NewNotFoundError(objectType, recordID)}
					} else {
						response.HTTPStatusCode = 200
						response.Body = record
					}
				}

			case "PATCH":
				if len(pathParts) > 1 {
					recordID := pathParts[1]
					err := r.store.UpdateRecord(objectType, recordID, body)
					if err != nil {
						response.HTTPStatusCode = 400
						response.Body = []sferrors.SalesforceError{{Message: err.Error(), ErrorCode: sferrors.ErrorCodeInvalidField}}
					} else {
						response.HTTPStatusCode = 204
						response.Body = nil
					}
				}

			case "DELETE":
				if len(pathParts) > 1 {
					recordID := pathParts[1]
					err := r.store.DeleteRecord(objectType, recordID)
					if err != nil {
						response.HTTPStatusCode = 404
						response.Body = []sferrors.SalesforceError{sferrors.NewNotFoundError(objectType, recordID)}
					} else {
						response.HTTPStatusCode = 204
						response.Body = nil
					}
				}
			}
		}
	} else if strings.Contains(url, "/query") {
		// Handle query requests
		queryStr := ""
		if idx := strings.Index(url, "?q="); idx != -1 {
			queryStr = url[idx+3:]
		}
		if queryStr != "" {
			records, err := r.executeSOQL(queryStr)
			if err != nil {
				response.HTTPStatusCode = 400
				response.Body = []sferrors.SalesforceError{sferrors.NewMalformedQueryError(err.Error())}
			} else {
				response.HTTPStatusCode = 200
				response.Body = QueryResponse{
					TotalSize: len(records),
					Done:      true,
					Records:   records,
				}
			}
		}
	}

	return response
}

// substituteReferences replaces @{refId.field} patterns in URLs
func substituteReferences(url string, refs map[string]interface{}) string {
	// Find all @{...} patterns
	result := url
	for refID, value := range refs {
		// Handle @{refId.Id} pattern
		if valueMap, ok := value.(map[string]interface{}); ok {
			if id, ok := valueMap["id"].(string); ok {
				result = strings.ReplaceAll(result, "@{"+refID+".id}", id)
				result = strings.ReplaceAll(result, "@{"+refID+".Id}", id)
			}
		}
		if resp, ok := value.(SObjectResponse); ok {
			result = strings.ReplaceAll(result, "@{"+refID+".id}", resp.ID)
			result = strings.ReplaceAll(result, "@{"+refID+".Id}", resp.ID)
		}
	}
	return result
}

// substituteBodyReferences replaces @{refId.field} patterns in request body
func substituteBodyReferences(body map[string]interface{}, refs map[string]interface{}) map[string]interface{} {
	if body == nil {
		return nil
	}

	result := make(map[string]interface{})
	for k, v := range body {
		if str, ok := v.(string); ok {
			// Check if it's a reference
			for refID, value := range refs {
				if strings.Contains(str, "@{"+refID) {
					if valueMap, ok := value.(map[string]interface{}); ok {
						if id, ok := valueMap["id"].(string); ok {
							str = strings.ReplaceAll(str, "@{"+refID+".id}", id)
							str = strings.ReplaceAll(str, "@{"+refID+".Id}", id)
						}
					}
					if resp, ok := value.(SObjectResponse); ok {
						str = strings.ReplaceAll(str, "@{"+refID+".id}", resp.ID)
						str = strings.ReplaceAll(str, "@{"+refID+".Id}", resp.ID)
					}
				}
			}
			result[k] = str
		} else {
			result[k] = v
		}
	}
	return result
}

// determineObjectType determines the object type from a Salesforce ID prefix
func determineObjectType(id string) string {
	if len(id) < 3 {
		return ""
	}
	prefix := id[:3]

	prefixMap := map[string]string{
		"001": "Account",
		"003": "Contact",
		"00Q": "Lead",
		"006": "Opportunity",
		"500": "Case",
		"005": "User",
		"00T": "Task",
		"00U": "Event",
	}

	return prefixMap[prefix]
}
