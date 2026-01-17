package storage

import (
	"fmt"
	"sync"
	"time"

	"github.com/MASA-JAPAN/go-salesforce-emulator/internal/idgen"
)

// MemoryStore is an in-memory implementation of the Store interface
type MemoryStore struct {
	mu sync.RWMutex

	// SObject data: objectType -> recordID -> Record
	records map[string]map[string]Record

	// Schema definitions: objectType -> SObjectDefinition
	schemas map[string]SObjectDefinition

	// Bulk jobs: jobID -> BulkJob
	bulkJobs map[string]*BulkJob

	// ID generators per object type
	idGenerators map[string]*idgen.Generator

	// Default user ID for system operations
	defaultUserID string
}

// NewMemoryStore creates a new in-memory store with standard objects registered
func NewMemoryStore() *MemoryStore {
	store := &MemoryStore{
		records:      make(map[string]map[string]Record),
		schemas:      make(map[string]SObjectDefinition),
		bulkJobs:     make(map[string]*BulkJob),
		idGenerators: make(map[string]*idgen.Generator),
	}

	// Register standard Salesforce objects
	for _, obj := range StandardSObjects {
		store.schemas[obj.Name] = obj
		store.records[obj.Name] = make(map[string]Record)
	}

	// Create a default user
	userGen := store.getIDGenerator("User")
	store.defaultUserID = userGen.Generate()
	store.records["User"][store.defaultUserID] = Record{
		"Id":               store.defaultUserID,
		"Username":         "admin@example.com",
		"FirstName":        "System",
		"LastName":         "Administrator",
		"Name":             "System Administrator",
		"Email":            "admin@example.com",
		"Alias":            "admin",
		"IsActive":         true,
		"CreatedDate":      time.Now().UTC().Format(time.RFC3339),
		"LastModifiedDate": time.Now().UTC().Format(time.RFC3339),
		"SystemModstamp":   time.Now().UTC().Format(time.RFC3339),
		"attributes": map[string]interface{}{
			"type": "User",
			"url":  fmt.Sprintf("/services/data/v58.0/sobjects/User/%s", store.defaultUserID),
		},
	}

	return store
}

func (s *MemoryStore) getIDGenerator(objectType string) *idgen.Generator {
	if gen, ok := s.idGenerators[objectType]; ok {
		return gen
	}
	gen := idgen.NewGenerator(objectType)
	s.idGenerators[objectType] = gen
	return gen
}

// CreateRecord creates a new record
func (s *MemoryStore) CreateRecord(objectType string, record Record) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if object type exists
	if _, ok := s.schemas[objectType]; !ok {
		return "", fmt.Errorf("object type not found: %s", objectType)
	}

	// Generate ID
	gen := s.getIDGenerator(objectType)
	id := gen.Generate()

	// Ensure records map exists for this object type
	if s.records[objectType] == nil {
		s.records[objectType] = make(map[string]Record)
	}

	// Create a copy of the record with system fields
	now := time.Now().UTC().Format(time.RFC3339)
	newRecord := make(Record)
	for k, v := range record {
		newRecord[k] = v
	}

	// Set system fields
	newRecord["Id"] = id
	newRecord["CreatedDate"] = now
	newRecord["CreatedById"] = s.defaultUserID
	newRecord["LastModifiedDate"] = now
	newRecord["LastModifiedById"] = s.defaultUserID
	newRecord["SystemModstamp"] = now
	newRecord["IsDeleted"] = false

	// Set attributes
	newRecord["attributes"] = map[string]interface{}{
		"type": objectType,
		"url":  fmt.Sprintf("/services/data/v58.0/sobjects/%s/%s", objectType, id),
	}

	// Handle Name field for Contact (computed from FirstName + LastName)
	if objectType == "Contact" || objectType == "Lead" || objectType == "User" {
		firstName, _ := newRecord["FirstName"].(string)
		lastName, _ := newRecord["LastName"].(string)
		if lastName != "" {
			if firstName != "" {
				newRecord["Name"] = firstName + " " + lastName
			} else {
				newRecord["Name"] = lastName
			}
		}
	}

	s.records[objectType][id] = newRecord

	return id, nil
}

// GetRecord retrieves a record by ID
func (s *MemoryStore) GetRecord(objectType, recordID string) (Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.schemas[objectType]; !ok {
		return nil, fmt.Errorf("object type not found: %s", objectType)
	}

	records, ok := s.records[objectType]
	if !ok {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	record, ok := records[recordID]
	if !ok {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	// Check if deleted
	if isDeleted, ok := record["IsDeleted"].(bool); ok && isDeleted {
		return nil, fmt.Errorf("record not found: %s", recordID)
	}

	return record, nil
}

// UpdateRecord updates an existing record
func (s *MemoryStore) UpdateRecord(objectType, recordID string, updates Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schemas[objectType]; !ok {
		return fmt.Errorf("object type not found: %s", objectType)
	}

	records, ok := s.records[objectType]
	if !ok {
		return fmt.Errorf("record not found: %s", recordID)
	}

	record, ok := records[recordID]
	if !ok {
		return fmt.Errorf("record not found: %s", recordID)
	}

	// Check if deleted
	if isDeleted, ok := record["IsDeleted"].(bool); ok && isDeleted {
		return fmt.Errorf("record not found: %s", recordID)
	}

	// Apply updates
	now := time.Now().UTC().Format(time.RFC3339)
	for k, v := range updates {
		// Skip read-only fields
		if k == "Id" || k == "CreatedDate" || k == "CreatedById" || k == "IsDeleted" || k == "attributes" {
			continue
		}
		record[k] = v
	}

	// Update system fields
	record["LastModifiedDate"] = now
	record["LastModifiedById"] = s.defaultUserID
	record["SystemModstamp"] = now

	// Update Name field for Contact/Lead/User
	if objectType == "Contact" || objectType == "Lead" || objectType == "User" {
		firstName, _ := record["FirstName"].(string)
		lastName, _ := record["LastName"].(string)
		if lastName != "" {
			if firstName != "" {
				record["Name"] = firstName + " " + lastName
			} else {
				record["Name"] = lastName
			}
		}
	}

	s.records[objectType][recordID] = record

	return nil
}

// DeleteRecord deletes a record (soft delete)
func (s *MemoryStore) DeleteRecord(objectType, recordID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.schemas[objectType]; !ok {
		return fmt.Errorf("object type not found: %s", objectType)
	}

	records, ok := s.records[objectType]
	if !ok {
		return fmt.Errorf("record not found: %s", recordID)
	}

	record, ok := records[recordID]
	if !ok {
		return fmt.Errorf("record not found: %s", recordID)
	}

	// Soft delete
	now := time.Now().UTC().Format(time.RFC3339)
	record["IsDeleted"] = true
	record["LastModifiedDate"] = now
	record["LastModifiedById"] = s.defaultUserID
	record["SystemModstamp"] = now

	s.records[objectType][recordID] = record

	return nil
}

// GetAllRecords returns all non-deleted records of a type
func (s *MemoryStore) GetAllRecords(objectType string) ([]Record, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, ok := s.schemas[objectType]; !ok {
		return nil, fmt.Errorf("object type not found: %s", objectType)
	}

	records, ok := s.records[objectType]
	if !ok {
		return []Record{}, nil
	}

	result := make([]Record, 0, len(records))
	for _, record := range records {
		// Skip deleted records
		if isDeleted, ok := record["IsDeleted"].(bool); ok && isDeleted {
			continue
		}
		result = append(result, record)
	}

	return result, nil
}

// CreateRecords creates multiple records
func (s *MemoryStore) CreateRecords(objectType string, records []Record) ([]CreateResult, error) {
	results := make([]CreateResult, len(records))

	for i, record := range records {
		id, err := s.CreateRecord(objectType, record)
		if err != nil {
			results[i] = CreateResult{
				ID:      "",
				Success: false,
				Errors:  []interface{}{err.Error()},
			}
		} else {
			results[i] = CreateResult{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			}
		}
	}

	return results, nil
}

// UpdateRecords updates multiple records
func (s *MemoryStore) UpdateRecords(objectType string, records []Record) ([]UpdateResult, error) {
	results := make([]UpdateResult, len(records))

	for i, record := range records {
		id, ok := record["Id"].(string)
		if !ok {
			results[i] = UpdateResult{
				ID:      "",
				Success: false,
				Errors:  []interface{}{"Id field is required"},
			}
			continue
		}

		err := s.UpdateRecord(objectType, id, record)
		if err != nil {
			results[i] = UpdateResult{
				ID:      id,
				Success: false,
				Errors:  []interface{}{err.Error()},
			}
		} else {
			results[i] = UpdateResult{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			}
		}
	}

	return results, nil
}

// DeleteRecords deletes multiple records
func (s *MemoryStore) DeleteRecords(objectType string, recordIDs []string) ([]DeleteResult, error) {
	results := make([]DeleteResult, len(recordIDs))

	for i, id := range recordIDs {
		err := s.DeleteRecord(objectType, id)
		if err != nil {
			results[i] = DeleteResult{
				ID:      id,
				Success: false,
				Errors:  []interface{}{err.Error()},
			}
		} else {
			results[i] = DeleteResult{
				ID:      id,
				Success: true,
				Errors:  []interface{}{},
			}
		}
	}

	return results, nil
}

// RegisterSObject registers a custom SObject definition
func (s *MemoryStore) RegisterSObject(definition SObjectDefinition) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.schemas[definition.Name] = definition
	if s.records[definition.Name] == nil {
		s.records[definition.Name] = make(map[string]Record)
	}

	return nil
}

// DescribeSObject returns the description of an SObject
func (s *MemoryStore) DescribeSObject(objectType string) (*SObjectDescription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	schema, ok := s.schemas[objectType]
	if !ok {
		return nil, fmt.Errorf("object type not found: %s", objectType)
	}

	return &SObjectDescription{
		SObjectDefinition: schema,
		URLs: map[string]string{
			"sobject":     fmt.Sprintf("/services/data/v58.0/sobjects/%s", objectType),
			"describe":    fmt.Sprintf("/services/data/v58.0/sobjects/%s/describe", objectType),
			"rowTemplate": fmt.Sprintf("/services/data/v58.0/sobjects/%s/{ID}", objectType),
		},
	}, nil
}

// DescribeGlobal returns a list of all SObjects
func (s *MemoryStore) DescribeGlobal() (*GlobalDescription, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sobjects := make([]SObjectInfo, 0, len(s.schemas))
	for name, schema := range s.schemas {
		sobjects = append(sobjects, SObjectInfo{
			Name:        name,
			Label:       schema.Label,
			LabelPlural: schema.LabelPlural,
			KeyPrefix:   schema.KeyPrefix,
			Custom:      schema.Custom,
			Createable:  schema.Createable,
			Updateable:  schema.Updateable,
			Deletable:   schema.Deletable,
			Queryable:   schema.Queryable,
			URLs: map[string]string{
				"sobject":  fmt.Sprintf("/services/data/v58.0/sobjects/%s", name),
				"describe": fmt.Sprintf("/services/data/v58.0/sobjects/%s/describe", name),
			},
		})
	}

	return &GlobalDescription{
		Encoding:     "UTF-8",
		MaxBatchSize: 200,
		SObjects:     sobjects,
	}, nil
}

// GetSObjectList returns a list of all registered SObject names
func (s *MemoryStore) GetSObjectList() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.schemas))
	for name := range s.schemas {
		names = append(names, name)
	}
	return names
}

// HasSObject checks if an SObject type is registered
func (s *MemoryStore) HasSObject(objectType string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.schemas[objectType]
	return ok
}

// CreateBulkJob creates a new bulk query job
func (s *MemoryStore) CreateBulkJob(config BulkJobConfig) (*BulkJob, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	gen := idgen.NewGeneratorWithPrefix("750") // Bulk job prefix
	jobID := gen.Generate()

	job := &BulkJob{
		ID:                     jobID,
		Operation:              config.Operation,
		Object:                 config.Object,
		State:                  JobStateUploadComplete,
		ContentType:            config.ContentType,
		CreatedDate:            time.Now().UTC(),
		CreatedById:            s.defaultUserID,
		SystemModstamp:         time.Now().UTC(),
		ConcurrencyMode:        "Parallel",
		ApiVersion:             58.0,
		JobType:                "V2Query",
		NumberRecordsProcessed: 0,
		Query:                  config.Query,
		Results:                []Record{},
		ResultLocators:         make(map[string]int),
	}

	s.bulkJobs[jobID] = job

	return job, nil
}

// GetBulkJob retrieves a bulk job by ID
func (s *MemoryStore) GetBulkJob(jobID string) (*BulkJob, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	job, ok := s.bulkJobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	return job, nil
}

// UpdateBulkJobState updates the state of a bulk job
func (s *MemoryStore) UpdateBulkJobState(jobID string, state JobState) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.bulkJobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.State = state
	job.SystemModstamp = time.Now().UTC()

	return nil
}

// SetBulkJobResults sets the results for a bulk job
func (s *MemoryStore) SetBulkJobResults(jobID string, results []Record) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.bulkJobs[jobID]
	if !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	job.Results = results
	job.NumberRecordsProcessed = len(results)

	return nil
}

// GetBulkJobResults retrieves paginated results from a bulk job
func (s *MemoryStore) GetBulkJobResults(jobID string, locator string, maxRecords int) (*BulkJobResults, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.bulkJobs[jobID]
	if !ok {
		return nil, "", fmt.Errorf("job not found: %s", jobID)
	}

	if job.State != JobStateJobComplete {
		return nil, "", fmt.Errorf("job not complete")
	}

	// Determine start index
	startIdx := 0
	if locator != "" {
		if idx, ok := job.ResultLocators[locator]; ok {
			startIdx = idx
		}
	}

	// Get page of results
	endIdx := startIdx + maxRecords
	if endIdx > len(job.Results) {
		endIdx = len(job.Results)
	}

	results := &BulkJobResults{
		Records: job.Results[startIdx:endIdx],
		Done:    endIdx >= len(job.Results),
	}

	// Generate next locator if there are more results
	nextLocator := ""
	if !results.Done {
		nextLocator = fmt.Sprintf("locator_%d", endIdx)
		job.ResultLocators[nextLocator] = endIdx
	}

	return results, nextLocator, nil
}

// DeleteBulkJob deletes a bulk job
func (s *MemoryStore) DeleteBulkJob(jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.bulkJobs[jobID]; !ok {
		return fmt.Errorf("job not found: %s", jobID)
	}

	delete(s.bulkJobs, jobID)
	return nil
}

// GetLimits returns API limits information
func (s *MemoryStore) GetLimits() *LimitsInfo {
	return &LimitsInfo{
		DailyApiRequests: LimitValue{
			Max:       100000,
			Remaining: 99000,
		},
		DailyAsyncApexExecutions: LimitValue{
			Max:       250000,
			Remaining: 250000,
		},
		DailyBulkApiRequests: LimitValue{
			Max:       10000,
			Remaining: 9990,
		},
		DailyBulkV2QueryJobs: LimitValue{
			Max:       10000,
			Remaining: 9990,
		},
		DataStorageMB: LimitValue{
			Max:       1024,
			Remaining: 900,
		},
		FileStorageMB: LimitValue{
			Max:       1024,
			Remaining: 950,
		},
		HourlyTimeBasedWorkflow: LimitValue{
			Max:       1000,
			Remaining: 1000,
		},
		SingleEmail: LimitValue{
			Max:       5000,
			Remaining: 5000,
		},
		StreamingApiConcurrentClients: LimitValue{
			Max:       2000,
			Remaining: 2000,
		},
	}
}

// GetRecordCounts returns record counts for specified object types
func (s *MemoryStore) GetRecordCounts(objectTypes []string) map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	counts := make(map[string]int)
	for _, objType := range objectTypes {
		records, ok := s.records[objType]
		if !ok {
			counts[objType] = 0
			continue
		}

		// Count non-deleted records
		count := 0
		for _, record := range records {
			if isDeleted, ok := record["IsDeleted"].(bool); !ok || !isDeleted {
				count++
			}
		}
		counts[objType] = count
	}

	return counts
}

// Reset clears all data from the store
func (s *MemoryStore) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clear all records but keep schemas
	for objType := range s.records {
		s.records[objType] = make(map[string]Record)
	}

	// Clear bulk jobs
	s.bulkJobs = make(map[string]*BulkJob)

	// Recreate default user
	userGen := s.getIDGenerator("User")
	s.defaultUserID = userGen.Generate()
	s.records["User"][s.defaultUserID] = Record{
		"Id":               s.defaultUserID,
		"Username":         "admin@example.com",
		"FirstName":        "System",
		"LastName":         "Administrator",
		"Name":             "System Administrator",
		"Email":            "admin@example.com",
		"Alias":            "admin",
		"IsActive":         true,
		"CreatedDate":      time.Now().UTC().Format(time.RFC3339),
		"LastModifiedDate": time.Now().UTC().Format(time.RFC3339),
		"SystemModstamp":   time.Now().UTC().Format(time.RFC3339),
		"attributes": map[string]interface{}{
			"type": "User",
			"url":  fmt.Sprintf("/services/data/v58.0/sobjects/User/%s", s.defaultUserID),
		},
	}
}

// GetDefaultUserID returns the default system user ID
func (s *MemoryStore) GetDefaultUserID() string {
	return s.defaultUserID
}
