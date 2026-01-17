package errors

import (
	"fmt"
)

// SalesforceError represents a Salesforce API error response
type SalesforceError struct {
	Message   string   `json:"message"`
	ErrorCode string   `json:"errorCode"`
	Fields    []string `json:"fields,omitempty"`
}

func (e SalesforceError) Error() string {
	return fmt.Sprintf("%s: %s", e.ErrorCode, e.Message)
}

// Common Salesforce error codes
const (
	ErrorCodeNotFound                = "NOT_FOUND"
	ErrorCodeInvalidField            = "INVALID_FIELD"
	ErrorCodeRequiredFieldMissing    = "REQUIRED_FIELD_MISSING"
	ErrorCodeDuplicateValue          = "DUPLICATE_VALUE"
	ErrorCodeMalformedQuery          = "MALFORMED_QUERY"
	ErrorCodeInvalidSessionID        = "INVALID_SESSION_ID"
	ErrorCodeInvalidGrant            = "invalid_grant"
	ErrorCodeJSONParserError         = "JSON_PARSER_ERROR"
	ErrorCodeInvalidQueryFilterOp    = "INVALID_QUERY_FILTER_OPERATOR"
	ErrorCodeEntityDeleted           = "ENTITY_IS_DELETED"
	ErrorCodeUnableToLockRow         = "UNABLE_TO_LOCK_ROW"
	ErrorCodeFieldIntegrity          = "FIELD_INTEGRITY_EXCEPTION"
	ErrorCodeInvalidType             = "INVALID_TYPE"
	ErrorCodeInvalidOperation        = "INVALID_OPERATION"
	ErrorCodeStringTooLong           = "STRING_TOO_LONG"
	ErrorCodeInvalidCrossReferenceKey = "INVALID_CROSS_REFERENCE_KEY"
	ErrorCodeUnsupportedGrantType    = "unsupported_grant_type"
	ErrorCodeMethodNotAllowed        = "METHOD_NOT_ALLOWED"
	ErrorCodeRequestLimitExceeded    = "REQUEST_LIMIT_EXCEEDED"
)

// NewNotFoundError creates a not found error
func NewNotFoundError(objectType, recordID string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("Provided external ID field does not exist or is not accessible: %s", recordID),
		ErrorCode: ErrorCodeNotFound,
		Fields:    []string{},
	}
}

// NewObjectNotFoundError creates an error for when an object type doesn't exist
func NewObjectNotFoundError(objectType string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("The requested resource does not exist"),
		ErrorCode: ErrorCodeNotFound,
		Fields:    []string{},
	}
}

// NewRequiredFieldError creates a required field missing error
func NewRequiredFieldError(fields ...string) SalesforceError {
	fieldList := ""
	for i, f := range fields {
		if i > 0 {
			fieldList += ", "
		}
		fieldList += f
	}
	return SalesforceError{
		Message:   fmt.Sprintf("Required fields are missing: [%s]", fieldList),
		ErrorCode: ErrorCodeRequiredFieldMissing,
		Fields:    fields,
	}
}

// NewMalformedQueryError creates a SOQL query error
func NewMalformedQueryError(details string) SalesforceError {
	return SalesforceError{
		Message:   details,
		ErrorCode: ErrorCodeMalformedQuery,
	}
}

// NewInvalidFieldError creates an invalid field error
func NewInvalidFieldError(fieldName, objectType string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("No such column '%s' on sobject of type %s", fieldName, objectType),
		ErrorCode: ErrorCodeInvalidField,
		Fields:    []string{fieldName},
	}
}

// NewInvalidSessionError creates an invalid session error
func NewInvalidSessionError() SalesforceError {
	return SalesforceError{
		Message:   "Session expired or invalid",
		ErrorCode: ErrorCodeInvalidSessionID,
	}
}

// NewJSONParserError creates a JSON parsing error
func NewJSONParserError(details string) SalesforceError {
	return SalesforceError{
		Message:   details,
		ErrorCode: ErrorCodeJSONParserError,
	}
}

// NewDuplicateValueError creates a duplicate value error
func NewDuplicateValueError(field, value string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("duplicate value found: %s duplicates value on record with id: <unknown>", field),
		ErrorCode: ErrorCodeDuplicateValue,
		Fields:    []string{field},
	}
}

// NewInvalidTypeError creates an invalid type error
func NewInvalidTypeError(objectType string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("sObject type '%s' is not supported.", objectType),
		ErrorCode: ErrorCodeInvalidType,
	}
}

// NewMethodNotAllowedError creates a method not allowed error
func NewMethodNotAllowedError(method string) SalesforceError {
	return SalesforceError{
		Message:   fmt.Sprintf("HTTP Method '%s' not allowed.", method),
		ErrorCode: ErrorCodeMethodNotAllowed,
	}
}

// NewRateLimitError creates a rate limit exceeded error
func NewRateLimitError() SalesforceError {
	return SalesforceError{
		Message:   "Request limit exceeded.",
		ErrorCode: ErrorCodeRequestLimitExceeded,
	}
}

// OAuthError represents an OAuth error response
type OAuthError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// NewOAuthError creates an OAuth error
func NewOAuthError(errorType, description string) OAuthError {
	return OAuthError{
		Error:            errorType,
		ErrorDescription: description,
	}
}
