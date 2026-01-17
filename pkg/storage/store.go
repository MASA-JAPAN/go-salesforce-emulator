package storage

import (
	"time"
)

// Record represents a generic Salesforce record
type Record map[string]interface{}

// Store defines the interface for the emulator's data storage
type Store interface {
	// SObject operations
	CreateRecord(objectType string, record Record) (string, error)
	GetRecord(objectType, recordID string) (Record, error)
	UpdateRecord(objectType, recordID string, updates Record) error
	DeleteRecord(objectType, recordID string) error
	GetAllRecords(objectType string) ([]Record, error)

	// Bulk operations
	CreateRecords(objectType string, records []Record) ([]CreateResult, error)
	UpdateRecords(objectType string, records []Record) ([]UpdateResult, error)
	DeleteRecords(objectType string, recordIDs []string) ([]DeleteResult, error)

	// Schema operations
	RegisterSObject(definition SObjectDefinition) error
	DescribeSObject(objectType string) (*SObjectDescription, error)
	DescribeGlobal() (*GlobalDescription, error)
	GetSObjectList() []string
	HasSObject(objectType string) bool

	// Bulk Job operations
	CreateBulkJob(config BulkJobConfig) (*BulkJob, error)
	GetBulkJob(jobID string) (*BulkJob, error)
	UpdateBulkJobState(jobID string, state JobState) error
	SetBulkJobResults(jobID string, results []Record) error
	GetBulkJobResults(jobID string, locator string, maxRecords int) (*BulkJobResults, string, error)
	DeleteBulkJob(jobID string) error

	// Limits
	GetLimits() *LimitsInfo
	GetRecordCounts(objectTypes []string) map[string]int

	// Utility
	Reset()
}

// CreateResult represents the result of a create operation
type CreateResult struct {
	ID      string        `json:"id"`
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

// UpdateResult represents the result of an update operation
type UpdateResult struct {
	ID      string        `json:"id"`
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

// DeleteResult represents the result of a delete operation
type DeleteResult struct {
	ID      string        `json:"id"`
	Success bool          `json:"success"`
	Errors  []interface{} `json:"errors"`
}

// SObjectDefinition defines the schema for an SObject
type SObjectDefinition struct {
	Name             string            `json:"name"`
	Label            string            `json:"label"`
	LabelPlural      string            `json:"labelPlural"`
	KeyPrefix        string            `json:"keyPrefix"`
	Custom           bool              `json:"custom"`
	Createable       bool              `json:"createable"`
	Updateable       bool              `json:"updateable"`
	Deletable        bool              `json:"deletable"`
	Queryable        bool              `json:"queryable"`
	Fields           []FieldDefinition `json:"fields"`
	RecordTypeInfos  []RecordTypeInfo  `json:"recordTypeInfos,omitempty"`
	ChildRelationships []ChildRelationship `json:"childRelationships,omitempty"`
}

// FieldDefinition defines a field on an SObject
type FieldDefinition struct {
	Name             string          `json:"name"`
	Label            string          `json:"label"`
	Type             FieldType       `json:"type"`
	Length           int             `json:"length,omitempty"`
	Precision        int             `json:"precision,omitempty"`
	Scale            int             `json:"scale,omitempty"`
	Nillable         bool            `json:"nillable"`
	Createable       bool            `json:"createable"`
	Updateable       bool            `json:"updateable"`
	Unique           bool            `json:"unique,omitempty"`
	ExternalId       bool            `json:"externalId,omitempty"`
	DefaultValue     interface{}     `json:"defaultValue,omitempty"`
	PicklistValues   []PicklistValue `json:"picklistValues,omitempty"`
	ReferenceTo      []string        `json:"referenceTo,omitempty"`
	RelationshipName string          `json:"relationshipName,omitempty"`
	SoapType         string          `json:"soapType,omitempty"`
}

// FieldType represents the type of a Salesforce field
type FieldType string

const (
	FieldTypeID           FieldType = "id"
	FieldTypeString       FieldType = "string"
	FieldTypeBoolean      FieldType = "boolean"
	FieldTypeInteger      FieldType = "int"
	FieldTypeDouble       FieldType = "double"
	FieldTypeCurrency     FieldType = "currency"
	FieldTypeDate         FieldType = "date"
	FieldTypeDatetime     FieldType = "datetime"
	FieldTypeTime         FieldType = "time"
	FieldTypeTextArea     FieldType = "textarea"
	FieldTypeLongTextArea FieldType = "long textarea"
	FieldTypeRichTextArea FieldType = "richtextarea"
	FieldTypePicklist     FieldType = "picklist"
	FieldTypeMultiPicklist FieldType = "multipicklist"
	FieldTypeReference    FieldType = "reference"
	FieldTypeEmail        FieldType = "email"
	FieldTypePhone        FieldType = "phone"
	FieldTypeURL          FieldType = "url"
	FieldTypePercent      FieldType = "percent"
	FieldTypeAddress      FieldType = "address"
	FieldTypeLocation     FieldType = "location"
	FieldTypeBase64       FieldType = "base64"
)

// PicklistValue represents a picklist option
type PicklistValue struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Active      bool   `json:"active"`
	DefaultValue bool   `json:"defaultValue"`
}

// RecordTypeInfo represents record type information
type RecordTypeInfo struct {
	RecordTypeId    string `json:"recordTypeId"`
	Name            string `json:"name"`
	Available       bool   `json:"available"`
	DefaultRecordTypeMapping bool `json:"defaultRecordTypeMapping"`
}

// ChildRelationship represents a child relationship
type ChildRelationship struct {
	ChildSObject       string `json:"childSObject"`
	Field              string `json:"field"`
	RelationshipName   string `json:"relationshipName"`
	CascadeDelete      bool   `json:"cascadeDelete"`
}

// SObjectDescription is the response for describe calls
type SObjectDescription struct {
	SObjectDefinition
	URLs map[string]string `json:"urls"`
}

// GlobalDescription is the response for describe global calls
type GlobalDescription struct {
	Encoding     string        `json:"encoding"`
	MaxBatchSize int           `json:"maxBatchSize"`
	SObjects     []SObjectInfo `json:"sobjects"`
}

// SObjectInfo is summary info for an sobject in global describe
type SObjectInfo struct {
	Name        string            `json:"name"`
	Label       string            `json:"label"`
	LabelPlural string            `json:"labelPlural"`
	KeyPrefix   string            `json:"keyPrefix"`
	Custom      bool              `json:"custom"`
	Createable  bool              `json:"createable"`
	Updateable  bool              `json:"updateable"`
	Deletable   bool              `json:"deletable"`
	Queryable   bool              `json:"queryable"`
	URLs        map[string]string `json:"urls"`
}

// BulkJobConfig is the configuration for creating a bulk job
type BulkJobConfig struct {
	Operation   string `json:"operation"`
	Object      string `json:"object"`
	Query       string `json:"query,omitempty"`
	ContentType string `json:"contentType,omitempty"`
}

// JobState represents the state of a bulk job
type JobState string

const (
	JobStateOpen           JobState = "Open"
	JobStateUploadComplete JobState = "UploadComplete"
	JobStateInProgress     JobState = "InProgress"
	JobStateJobComplete    JobState = "JobComplete"
	JobStateAborted        JobState = "Aborted"
	JobStateFailed         JobState = "Failed"
)

// BulkJob represents a bulk API job
type BulkJob struct {
	ID                     string    `json:"id"`
	Operation              string    `json:"operation"`
	Object                 string    `json:"object"`
	State                  JobState  `json:"state"`
	ContentType            string    `json:"contentType"`
	CreatedDate            time.Time `json:"createdDate"`
	CreatedById            string    `json:"createdById"`
	SystemModstamp         time.Time `json:"systemModstamp"`
	ConcurrencyMode        string    `json:"concurrencyMode"`
	ApiVersion             float64   `json:"apiVersion"`
	JobType                string    `json:"jobType"`
	NumberRecordsProcessed int       `json:"numberRecordsProcessed"`
	Query                  string    `json:"query,omitempty"`

	// Internal fields (not serialized)
	Results         []Record          `json:"-"`
	ResultLocators  map[string]int    `json:"-"`
}

// BulkJobResults represents paginated bulk job results
type BulkJobResults struct {
	Records []Record
	Done    bool
}

// LimitsInfo represents API limits information
type LimitsInfo struct {
	DailyApiRequests        LimitValue `json:"DailyApiRequests"`
	DailyAsyncApexExecutions LimitValue `json:"DailyAsyncApexExecutions"`
	DailyBulkApiRequests    LimitValue `json:"DailyBulkApiRequests"`
	DailyBulkV2QueryJobs    LimitValue `json:"DailyBulkV2QueryJobs"`
	DataStorageMB           LimitValue `json:"DataStorageMB"`
	FileStorageMB           LimitValue `json:"FileStorageMB"`
	HourlyTimeBasedWorkflow LimitValue `json:"HourlyTimeBasedWorkflow"`
	SingleEmail             LimitValue `json:"SingleEmail"`
	StreamingApiConcurrentClients LimitValue `json:"StreamingApiConcurrentClients"`
}

// LimitValue represents a single limit with max and remaining
type LimitValue struct {
	Max       int `json:"Max"`
	Remaining int `json:"Remaining"`
}
