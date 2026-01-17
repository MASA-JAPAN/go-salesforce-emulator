package testutil

import (
	"fmt"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// Fixtures helps set up test data
type Fixtures struct {
	store *storage.MemoryStore
}

// NewFixtures creates a new fixtures helper
func NewFixtures(store *storage.MemoryStore) *Fixtures {
	return &Fixtures{store: store}
}

// LoadSampleAccounts creates sample account records
func (f *Fixtures) LoadSampleAccounts(count int) ([]string, error) {
	ids := make([]string, count)

	industries := []string{"Technology", "Healthcare", "Finance", "Manufacturing", "Consulting", "Education"}

	for i := 0; i < count; i++ {
		record := NewAccountBuilder().
			WithName(fmt.Sprintf("Test Account %d", i+1)).
			WithIndustry(industries[i%len(industries)]).
			WithWebsite(fmt.Sprintf("https://account%d.example.com", i+1)).
			WithPhone(fmt.Sprintf("+1-555-%04d", i+1)).
			WithAnnualRevenue(float64((i + 1) * 100000)).
			WithNumberOfEmployees((i + 1) * 10).
			Build()

		id, err := f.store.CreateRecord("Account", record)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

// LoadSampleContacts creates sample contact records
func (f *Fixtures) LoadSampleContacts(count int, accountIds []string) ([]string, error) {
	ids := make([]string, count)

	firstNames := []string{"John", "Jane", "Bob", "Alice", "Charlie", "Diana", "Edward", "Fiona"}
	lastNames := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis"}
	titles := []string{"CEO", "CTO", "VP Sales", "Manager", "Director", "Engineer", "Analyst", "Consultant"}

	for i := 0; i < count; i++ {
		builder := NewContactBuilder().
			WithFirstName(firstNames[i%len(firstNames)]).
			WithLastName(lastNames[i%len(lastNames)]).
			WithEmail(fmt.Sprintf("contact%d@example.com", i+1)).
			WithPhone(fmt.Sprintf("+1-555-%04d", 1000+i+1)).
			WithTitle(titles[i%len(titles)])

		if len(accountIds) > 0 {
			builder.WithAccountId(accountIds[i%len(accountIds)])
		}

		id, err := f.store.CreateRecord("Contact", builder.Build())
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

// LoadSampleLeads creates sample lead records
func (f *Fixtures) LoadSampleLeads(count int) ([]string, error) {
	ids := make([]string, count)

	firstNames := []string{"Michael", "Sarah", "David", "Emily", "James", "Lisa"}
	lastNames := []string{"Anderson", "Taylor", "Thomas", "Jackson", "White", "Harris"}
	companies := []string{"Acme Corp", "Globex", "Initech", "Umbrella Corp", "Wayne Enterprises", "Stark Industries"}

	for i := 0; i < count; i++ {
		record := NewLeadBuilder().
			WithFirstName(firstNames[i%len(firstNames)]).
			WithLastName(lastNames[i%len(lastNames)]).
			WithCompany(companies[i%len(companies)]).
			WithEmail(fmt.Sprintf("lead%d@example.com", i+1)).
			WithPhone(fmt.Sprintf("+1-555-%04d", 2000+i+1)).
			WithStatus("Open - Not Contacted").
			Build()

		id, err := f.store.CreateRecord("Lead", record)
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

// LoadSampleOpportunities creates sample opportunity records
func (f *Fixtures) LoadSampleOpportunities(count int, accountIds []string) ([]string, error) {
	ids := make([]string, count)

	stages := []string{"Prospecting", "Qualification", "Needs Analysis", "Value Proposition", "Proposal/Price Quote", "Negotiation/Review"}

	for i := 0; i < count; i++ {
		builder := NewOpportunityBuilder().
			WithName(fmt.Sprintf("Opportunity %d", i+1)).
			WithAmount(float64((i + 1) * 50000)).
			WithCloseDate("2025-12-31").
			WithStageName(stages[i%len(stages)]).
			WithProbability(float64((i%6 + 1) * 15))

		if len(accountIds) > 0 {
			builder.WithAccountId(accountIds[i%len(accountIds)])
		}

		id, err := f.store.CreateRecord("Opportunity", builder.Build())
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

// LoadSampleCases creates sample case records
func (f *Fixtures) LoadSampleCases(count int, accountIds, contactIds []string) ([]string, error) {
	ids := make([]string, count)

	subjects := []string{"Login Issue", "Feature Request", "Bug Report", "Billing Question", "Technical Support", "General Inquiry"}
	statuses := []string{"New", "Working", "Escalated", "Closed"}
	priorities := []string{"Low", "Medium", "High"}
	origins := []string{"Phone", "Email", "Web"}

	for i := 0; i < count; i++ {
		builder := NewCaseBuilder().
			WithSubject(fmt.Sprintf("%s #%d", subjects[i%len(subjects)], i+1)).
			WithDescription(fmt.Sprintf("Description for case %d", i+1)).
			WithStatus(statuses[i%len(statuses)]).
			WithPriority(priorities[i%len(priorities)]).
			WithOrigin(origins[i%len(origins)])

		if len(accountIds) > 0 {
			builder.WithAccountId(accountIds[i%len(accountIds)])
		}
		if len(contactIds) > 0 {
			builder.WithContactId(contactIds[i%len(contactIds)])
		}

		id, err := f.store.CreateRecord("Case", builder.Build())
		if err != nil {
			return nil, err
		}
		ids[i] = id
	}

	return ids, nil
}

// LoadBasicCRMData creates a basic CRM dataset
func (f *Fixtures) LoadBasicCRMData() error {
	// Create 10 accounts
	accountIds, err := f.LoadSampleAccounts(10)
	if err != nil {
		return fmt.Errorf("failed to create accounts: %w", err)
	}

	// Create 50 contacts (5 per account)
	_, err = f.LoadSampleContacts(50, accountIds)
	if err != nil {
		return fmt.Errorf("failed to create contacts: %w", err)
	}

	// Create 20 leads
	_, err = f.LoadSampleLeads(20)
	if err != nil {
		return fmt.Errorf("failed to create leads: %w", err)
	}

	// Create 30 opportunities
	_, err = f.LoadSampleOpportunities(30, accountIds)
	if err != nil {
		return fmt.Errorf("failed to create opportunities: %w", err)
	}

	return nil
}

// LoadHighVolumeData creates a large dataset for performance testing
func (f *Fixtures) LoadHighVolumeData() error {
	// Create 100 accounts
	accountIds, err := f.LoadSampleAccounts(100)
	if err != nil {
		return fmt.Errorf("failed to create accounts: %w", err)
	}

	// Create 500 contacts
	contactIds, err := f.LoadSampleContacts(500, accountIds)
	if err != nil {
		return fmt.Errorf("failed to create contacts: %w", err)
	}

	// Create 200 leads
	_, err = f.LoadSampleLeads(200)
	if err != nil {
		return fmt.Errorf("failed to create leads: %w", err)
	}

	// Create 100 opportunities
	_, err = f.LoadSampleOpportunities(100, accountIds)
	if err != nil {
		return fmt.Errorf("failed to create opportunities: %w", err)
	}

	// Create 200 cases
	_, err = f.LoadSampleCases(200, accountIds, contactIds)
	if err != nil {
		return fmt.Errorf("failed to create cases: %w", err)
	}

	return nil
}

// Scenario represents a pre-configured test scenario
type Scenario struct {
	Name        string
	Description string
	Setup       func(*emulator.Emulator) error
}

// EmptyOrgScenario creates an empty organization
var EmptyOrgScenario = Scenario{
	Name:        "empty_org",
	Description: "Empty organization with no data",
	Setup: func(e *emulator.Emulator) error {
		e.Reset()
		return nil
	},
}

// BasicCRMScenario creates a basic CRM dataset
var BasicCRMScenario = Scenario{
	Name:        "basic_crm",
	Description: "Basic CRM data with accounts, contacts, leads, and opportunities",
	Setup: func(e *emulator.Emulator) error {
		fixtures := NewFixtures(e.Store())
		return fixtures.LoadBasicCRMData()
	},
}

// HighVolumeScenario creates a large dataset
var HighVolumeScenario = Scenario{
	Name:        "high_volume",
	Description: "Large dataset for pagination and performance testing",
	Setup: func(e *emulator.Emulator) error {
		fixtures := NewFixtures(e.Store())
		return fixtures.LoadHighVolumeData()
	},
}

// AvailableScenarios lists all pre-built scenarios
var AvailableScenarios = map[string]Scenario{
	"empty_org":   EmptyOrgScenario,
	"basic_crm":   BasicCRMScenario,
	"high_volume": HighVolumeScenario,
}

// LoadScenario loads a pre-built scenario into the emulator
func LoadScenario(e *emulator.Emulator, scenarioName string) error {
	scenario, ok := AvailableScenarios[scenarioName]
	if !ok {
		return fmt.Errorf("scenario not found: %s", scenarioName)
	}
	return scenario.Setup(e)
}
