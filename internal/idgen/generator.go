package idgen

import (
	"crypto/rand"
	"encoding/base32"
	"strings"
	"sync"
)

// Generator generates Salesforce-style 18-character IDs
type Generator struct {
	mu      sync.Mutex
	prefix  string
	counter uint64
}

// Standard Salesforce key prefixes for common objects
var StandardPrefixes = map[string]string{
	"Account":          "001",
	"Contact":          "003",
	"Lead":             "00Q",
	"Opportunity":      "006",
	"Case":             "500",
	"User":             "005",
	"Task":             "00T",
	"Event":            "00U",
	"Campaign":         "701",
	"Contract":         "800",
	"Product2":         "01t",
	"Pricebook2":       "01s",
	"PricebookEntry":   "01u",
	"Asset":            "02i",
	"Order":            "801",
	"OrderItem":        "802",
	"Quote":            "0Q0",
	"QuoteLineItem":    "0QL",
	"ContentDocument":  "069",
	"ContentVersion":   "068",
	"Attachment":       "00P",
	"Note":             "002",
	"EmailMessage":     "02s",
	"FeedItem":         "0D5",
	"Group":            "00G",
	"GroupMember":      "011",
	"CampaignMember":   "00v",
	"OpportunityLineItem": "00k",
	"AccountContactRole": "00J",
	"OpportunityContactRole": "00K",
	"CaseComment":      "00a",
	"Solution":         "501",
	"Report":           "00O",
	"Dashboard":        "01Z",
	"Document":         "015",
	"Folder":           "00l",
	"ApexClass":        "01p",
	"ApexTrigger":      "01q",
	"CustomField":      "00N",
	"CustomObject":     "01I",
}

// NewGenerator creates a new ID generator for a specific object type
func NewGenerator(objectType string) *Generator {
	prefix := StandardPrefixes[objectType]
	if prefix == "" {
		// For custom objects or unknown types, use a generic prefix
		prefix = "a0"
		// Add first letter of object type
		if len(objectType) > 0 {
			prefix += string(objectType[0])
		} else {
			prefix += "0"
		}
	}
	return &Generator{
		prefix: prefix,
	}
}

// NewGeneratorWithPrefix creates a generator with a custom prefix
func NewGeneratorWithPrefix(prefix string) *Generator {
	return &Generator{
		prefix: prefix,
	}
}

// Generate creates a new unique 18-character Salesforce ID
func (g *Generator) Generate() string {
	g.mu.Lock()
	g.counter++
	g.mu.Unlock()

	// Generate random bytes for uniqueness
	randomBytes := make([]byte, 10)
	rand.Read(randomBytes)

	// Create base ID (15 chars)
	// Format: 3-char prefix + 12-char unique portion
	encoded := base32.StdEncoding.EncodeToString(randomBytes)
	encoded = strings.ToUpper(encoded)

	// Take 12 characters from encoded (ensuring alphanumeric)
	uniquePart := ""
	for _, c := range encoded {
		if (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			uniquePart += string(c)
			if len(uniquePart) >= 12 {
				break
			}
		}
	}

	// Pad if needed
	for len(uniquePart) < 12 {
		uniquePart += "0"
	}

	base15 := g.prefix + uniquePart[:12]

	// Calculate 3-character checksum suffix for 18-character ID
	suffix := calculateChecksum(base15)

	return base15 + suffix
}

// calculateChecksum calculates the 3-character case-insensitive suffix
// This makes Salesforce IDs case-insensitive
func calculateChecksum(base15 string) string {
	if len(base15) != 15 {
		return "AAA"
	}

	lookup := "ABCDEFGHIJKLMNOPQRSTUVWXYZ012345"
	suffix := ""

	// Process 3 groups of 5 characters
	for i := 0; i < 3; i++ {
		flags := 0
		for j := 0; j < 5; j++ {
			c := base15[i*5+j]
			if c >= 'A' && c <= 'Z' {
				flags += 1 << j
			}
		}
		suffix += string(lookup[flags])
	}

	return suffix
}

// IsValid checks if an ID appears to be a valid Salesforce ID format
func IsValid(id string) bool {
	if len(id) != 15 && len(id) != 18 {
		return false
	}

	// Check all characters are alphanumeric
	for _, c := range id {
		if !((c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') || (c >= '0' && c <= '9')) {
			return false
		}
	}

	return true
}

// Normalize converts a 15-character ID to 18-character format
func Normalize(id string) string {
	if len(id) == 18 {
		return id
	}
	if len(id) != 15 {
		return id
	}
	return id + calculateChecksum(id)
}

// GetPrefix extracts the 3-character prefix from an ID
func GetPrefix(id string) string {
	if len(id) < 3 {
		return ""
	}
	return id[:3]
}

// GetObjectType attempts to determine the object type from an ID prefix
func GetObjectType(id string) string {
	prefix := GetPrefix(id)
	for objType, p := range StandardPrefixes {
		if p == prefix {
			return objType
		}
	}
	return ""
}
