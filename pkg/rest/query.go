package rest

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// QueryResponse represents a SOQL query response
type QueryResponse struct {
	TotalSize      int              `json:"totalSize"`
	Done           bool             `json:"done"`
	NextRecordsURL string           `json:"nextRecordsUrl,omitempty"`
	Records        []storage.Record `json:"records"`
}

// queryState holds the state for paginated queries
type queryState struct {
	records    []storage.Record
	startIndex int
}

var (
	queryStates = make(map[string]*queryState)
	queryIDGen  = 0
)

// handleQuery handles GET /services/data/vXX.X/query?q=...
func (r *Router) handleQuery(w http.ResponseWriter, req *http.Request, params []string) {
	query := req.URL.Query().Get("q")
	if query == "" {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMalformedQueryError("No query string provided"),
		}, http.StatusBadRequest)
		return
	}

	// Parse and execute the query
	records, err := r.executeSOQL(query)
	if err != nil {
		r.respondError(w, []sferrors.SalesforceError{
			sferrors.NewMalformedQueryError(err.Error()),
		}, http.StatusBadRequest)
		return
	}

	// Determine batch size (default 2000)
	batchSize := 2000

	// Check for Sforce-Query-Options header for batch size
	if options := req.Header.Get("Sforce-Query-Options"); options != "" {
		if size := parseBatchSize(options); size > 0 {
			batchSize = size
		}
	}

	response := QueryResponse{
		TotalSize: len(records),
		Records:   records,
		Done:      true,
	}

	// If more records than batch size, paginate
	if len(records) > batchSize {
		response.Records = records[:batchSize]
		response.Done = false

		// Store state for pagination
		queryIDGen++
		locator := fmt.Sprintf("query-%d", queryIDGen)
		queryStates[locator] = &queryState{
			records:    records,
			startIndex: batchSize,
		}
		response.NextRecordsURL = fmt.Sprintf("/services/data/v%s/query/%s", r.apiVersion, locator)
	}

	r.respondJSON(w, response, http.StatusOK)
}

// handleQueryMore handles GET /services/data/vXX.X/query/{locator}
func (r *Router) handleQueryMore(w http.ResponseWriter, req *http.Request, params []string) {
	locator := params[0]

	state, ok := queryStates[locator]
	if !ok {
		r.respondError(w, []sferrors.SalesforceError{
			{Message: "Invalid query locator", ErrorCode: sferrors.ErrorCodeNotFound},
		}, http.StatusNotFound)
		return
	}

	batchSize := 2000
	if options := req.Header.Get("Sforce-Query-Options"); options != "" {
		if size := parseBatchSize(options); size > 0 {
			batchSize = size
		}
	}

	endIndex := state.startIndex + batchSize
	if endIndex > len(state.records) {
		endIndex = len(state.records)
	}

	response := QueryResponse{
		TotalSize: len(state.records),
		Records:   state.records[state.startIndex:endIndex],
		Done:      endIndex >= len(state.records),
	}

	if !response.Done {
		state.startIndex = endIndex
		response.NextRecordsURL = fmt.Sprintf("/services/data/v%s/query/%s", r.apiVersion, locator)
	} else {
		delete(queryStates, locator)
	}

	r.respondJSON(w, response, http.StatusOK)
}

// executeSOQL parses and executes a SOQL query
func (r *Router) executeSOQL(query string) ([]storage.Record, error) {
	// Simple SOQL parser - handles basic SELECT ... FROM ... WHERE ... ORDER BY ... LIMIT
	query = strings.TrimSpace(query)

	// Parse SELECT clause
	selectMatch := regexp.MustCompile(`(?i)^SELECT\s+(.+?)\s+FROM\s+`).FindStringSubmatch(query)
	if selectMatch == nil {
		return nil, fmt.Errorf("invalid SOQL: missing SELECT or FROM clause")
	}
	fields := parseSelectFields(selectMatch[1])

	// Parse FROM clause
	fromMatch := regexp.MustCompile(`(?i)FROM\s+(\w+)`).FindStringSubmatch(query)
	if fromMatch == nil {
		return nil, fmt.Errorf("invalid SOQL: missing FROM clause")
	}
	objectType := fromMatch[1]

	// Check if object exists
	if !r.store.HasSObject(objectType) {
		return nil, fmt.Errorf("sObject type '%s' is not supported", objectType)
	}

	// Get all records
	allRecords, err := r.store.GetAllRecords(objectType)
	if err != nil {
		return nil, err
	}

	// Apply WHERE clause if present
	whereMatch := regexp.MustCompile(`(?i)WHERE\s+(.+?)(?:\s+ORDER\s+BY|\s+LIMIT|\s+OFFSET|\s*$)`).FindStringSubmatch(query)
	if whereMatch != nil {
		allRecords = filterRecords(allRecords, whereMatch[1])
	}

	// Apply ORDER BY if present
	orderMatch := regexp.MustCompile(`(?i)ORDER\s+BY\s+(\w+)(?:\s+(ASC|DESC))?`).FindStringSubmatch(query)
	if orderMatch != nil {
		allRecords = sortRecords(allRecords, orderMatch[1], strings.ToUpper(orderMatch[2]) == "DESC")
	}

	// Apply LIMIT if present
	limitMatch := regexp.MustCompile(`(?i)LIMIT\s+(\d+)`).FindStringSubmatch(query)
	if limitMatch != nil {
		limit, _ := strconv.Atoi(limitMatch[1])
		if limit < len(allRecords) {
			allRecords = allRecords[:limit]
		}
	}

	// Apply OFFSET if present
	offsetMatch := regexp.MustCompile(`(?i)OFFSET\s+(\d+)`).FindStringSubmatch(query)
	if offsetMatch != nil {
		offset, _ := strconv.Atoi(offsetMatch[1])
		if offset < len(allRecords) {
			allRecords = allRecords[offset:]
		} else {
			allRecords = []storage.Record{}
		}
	}

	// Project fields
	result := make([]storage.Record, len(allRecords))
	for i, record := range allRecords {
		result[i] = projectFields(record, fields, objectType)
	}

	return result, nil
}

// parseSelectFields parses the SELECT field list
func parseSelectFields(fieldsStr string) []string {
	var fields []string
	for _, f := range strings.Split(fieldsStr, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// projectFields creates a new record with only the selected fields
func projectFields(record storage.Record, fields []string, objectType string) storage.Record {
	result := make(storage.Record)

	// Always include attributes
	result["attributes"] = map[string]interface{}{
		"type": objectType,
		"url":  record["attributes"].(map[string]interface{})["url"],
	}

	for _, field := range fields {
		// Handle aggregate functions
		if strings.HasPrefix(strings.ToUpper(field), "COUNT(") {
			// For COUNT(), we need special handling at the query level
			continue
		}

		// Handle relationship fields (e.g., Account.Name)
		if strings.Contains(field, ".") {
			// For now, skip relationship fields in projection
			continue
		}

		if val, ok := record[field]; ok {
			result[field] = val
		} else {
			result[field] = nil
		}
	}

	return result
}

// filterRecords applies WHERE clause filtering
func filterRecords(records []storage.Record, whereClause string) []storage.Record {
	var result []storage.Record

	// Parse simple conditions (field = 'value', field != 'value', field = number, etc.)
	conditions := parseWhereConditions(whereClause)

	for _, record := range records {
		if matchesConditions(record, conditions) {
			result = append(result, record)
		}
	}

	return result
}

type condition struct {
	field    string
	operator string
	value    interface{}
}

// parseWhereConditions parses WHERE clause into conditions
func parseWhereConditions(whereClause string) []condition {
	var conditions []condition

	// Handle AND conditions
	parts := regexp.MustCompile(`(?i)\s+AND\s+`).Split(whereClause, -1)

	for _, part := range parts {
		part = strings.TrimSpace(part)

		// Match: field = 'value'
		if match := regexp.MustCompile(`(\w+)\s*(=|!=|<>|<|>|<=|>=|LIKE)\s*'([^']*)'`).FindStringSubmatch(part); match != nil {
			conditions = append(conditions, condition{
				field:    match[1],
				operator: strings.ToUpper(match[2]),
				value:    match[3],
			})
			continue
		}

		// Match: field = number
		if match := regexp.MustCompile(`(\w+)\s*(=|!=|<>|<|>|<=|>=)\s*(\d+(?:\.\d+)?)`).FindStringSubmatch(part); match != nil {
			val, _ := strconv.ParseFloat(match[3], 64)
			conditions = append(conditions, condition{
				field:    match[1],
				operator: match[2],
				value:    val,
			})
			continue
		}

		// Match: field = true/false
		if match := regexp.MustCompile(`(?i)(\w+)\s*(=|!=)\s*(true|false)`).FindStringSubmatch(part); match != nil {
			conditions = append(conditions, condition{
				field:    match[1],
				operator: match[2],
				value:    strings.ToLower(match[3]) == "true",
			})
			continue
		}

		// Match: field = null
		if match := regexp.MustCompile(`(?i)(\w+)\s*(=|!=)\s*null`).FindStringSubmatch(part); match != nil {
			conditions = append(conditions, condition{
				field:    match[1],
				operator: match[2],
				value:    nil,
			})
			continue
		}

		// Match: field IN ('val1', 'val2', ...)
		if match := regexp.MustCompile(`(?i)(\w+)\s+IN\s*\(([^)]+)\)`).FindStringSubmatch(part); match != nil {
			values := parseInValues(match[2])
			conditions = append(conditions, condition{
				field:    match[1],
				operator: "IN",
				value:    values,
			})
			continue
		}
	}

	return conditions
}

// parseInValues parses the values in an IN clause
func parseInValues(valuesStr string) []string {
	var values []string
	for _, v := range strings.Split(valuesStr, ",") {
		v = strings.TrimSpace(v)
		v = strings.Trim(v, "'\"")
		values = append(values, v)
	}
	return values
}

// matchesConditions checks if a record matches all conditions
func matchesConditions(record storage.Record, conditions []condition) bool {
	for _, cond := range conditions {
		val := record[cond.field]

		switch cond.operator {
		case "=":
			if !equals(val, cond.value) {
				return false
			}
		case "!=", "<>":
			if equals(val, cond.value) {
				return false
			}
		case "<":
			if !lessThan(val, cond.value) {
				return false
			}
		case ">":
			if !greaterThan(val, cond.value) {
				return false
			}
		case "<=":
			if !lessThanOrEqual(val, cond.value) {
				return false
			}
		case ">=":
			if !greaterThanOrEqual(val, cond.value) {
				return false
			}
		case "LIKE":
			if !matchesLike(val, cond.value.(string)) {
				return false
			}
		case "IN":
			if !inValues(val, cond.value.([]string)) {
				return false
			}
		}
	}
	return true
}

func equals(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Convert to strings for comparison
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}

func lessThan(a, b interface{}) bool {
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if aOk && bOk {
		return aNum < bNum
	}
	return fmt.Sprintf("%v", a) < fmt.Sprintf("%v", b)
}

func greaterThan(a, b interface{}) bool {
	aNum, aOk := toFloat(a)
	bNum, bOk := toFloat(b)
	if aOk && bOk {
		return aNum > bNum
	}
	return fmt.Sprintf("%v", a) > fmt.Sprintf("%v", b)
}

func lessThanOrEqual(a, b interface{}) bool {
	return equals(a, b) || lessThan(a, b)
}

func greaterThanOrEqual(a, b interface{}) bool {
	return equals(a, b) || greaterThan(a, b)
}

func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	}
	return 0, false
}

func matchesLike(val interface{}, pattern string) bool {
	if val == nil {
		return false
	}
	valStr := fmt.Sprintf("%v", val)

	// Convert SQL LIKE pattern to regex
	pattern = strings.ReplaceAll(pattern, "%", ".*")
	pattern = strings.ReplaceAll(pattern, "_", ".")
	pattern = "^" + pattern + "$"

	matched, _ := regexp.MatchString("(?i)"+pattern, valStr)
	return matched
}

func inValues(val interface{}, values []string) bool {
	if val == nil {
		return false
	}
	valStr := fmt.Sprintf("%v", val)
	for _, v := range values {
		if valStr == v {
			return true
		}
	}
	return false
}

// sortRecords sorts records by a field
func sortRecords(records []storage.Record, field string, descending bool) []storage.Record {
	result := make([]storage.Record, len(records))
	copy(result, records)

	// Simple bubble sort for now
	for i := 0; i < len(result)-1; i++ {
		for j := 0; j < len(result)-i-1; j++ {
			aVal := result[j][field]
			bVal := result[j+1][field]

			shouldSwap := false
			if descending {
				shouldSwap = greaterThan(bVal, aVal)
			} else {
				shouldSwap = greaterThan(aVal, bVal)
			}

			if shouldSwap {
				result[j], result[j+1] = result[j+1], result[j]
			}
		}
	}

	return result
}

// parseBatchSize extracts batchSize from Sforce-Query-Options header
func parseBatchSize(options string) int {
	match := regexp.MustCompile(`batchSize=(\d+)`).FindStringSubmatch(options)
	if match != nil {
		size, _ := strconv.Atoi(match[1])
		return size
	}
	return 0
}
