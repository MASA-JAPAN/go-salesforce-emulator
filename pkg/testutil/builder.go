package testutil

import (
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// RecordBuilder helps build test records
type RecordBuilder struct {
	objectType string
	record     storage.Record
}

// NewRecordBuilder creates a new record builder
func NewRecordBuilder(objectType string) *RecordBuilder {
	return &RecordBuilder{
		objectType: objectType,
		record: storage.Record{
			"attributes": map[string]interface{}{
				"type": objectType,
			},
		},
	}
}

// Set sets a field value
func (b *RecordBuilder) Set(field string, value interface{}) *RecordBuilder {
	b.record[field] = value
	return b
}

// Build returns the built record
func (b *RecordBuilder) Build() storage.Record {
	return b.record
}

// AccountBuilder helps build Account records
type AccountBuilder struct {
	*RecordBuilder
}

// NewAccountBuilder creates a new Account builder
func NewAccountBuilder() *AccountBuilder {
	return &AccountBuilder{
		RecordBuilder: NewRecordBuilder("Account"),
	}
}

// WithName sets the account name
func (b *AccountBuilder) WithName(name string) *AccountBuilder {
	b.Set("Name", name)
	return b
}

// WithIndustry sets the industry
func (b *AccountBuilder) WithIndustry(industry string) *AccountBuilder {
	b.Set("Industry", industry)
	return b
}

// WithWebsite sets the website
func (b *AccountBuilder) WithWebsite(website string) *AccountBuilder {
	b.Set("Website", website)
	return b
}

// WithPhone sets the phone
func (b *AccountBuilder) WithPhone(phone string) *AccountBuilder {
	b.Set("Phone", phone)
	return b
}

// WithAnnualRevenue sets the annual revenue
func (b *AccountBuilder) WithAnnualRevenue(revenue float64) *AccountBuilder {
	b.Set("AnnualRevenue", revenue)
	return b
}

// WithNumberOfEmployees sets the number of employees
func (b *AccountBuilder) WithNumberOfEmployees(count int) *AccountBuilder {
	b.Set("NumberOfEmployees", count)
	return b
}

// WithBillingAddress sets billing address fields
func (b *AccountBuilder) WithBillingAddress(street, city, state, postalCode, country string) *AccountBuilder {
	b.Set("BillingStreet", street)
	b.Set("BillingCity", city)
	b.Set("BillingState", state)
	b.Set("BillingPostalCode", postalCode)
	b.Set("BillingCountry", country)
	return b
}

// ContactBuilder helps build Contact records
type ContactBuilder struct {
	*RecordBuilder
}

// NewContactBuilder creates a new Contact builder
func NewContactBuilder() *ContactBuilder {
	return &ContactBuilder{
		RecordBuilder: NewRecordBuilder("Contact"),
	}
}

// WithFirstName sets the first name
func (b *ContactBuilder) WithFirstName(firstName string) *ContactBuilder {
	b.Set("FirstName", firstName)
	return b
}

// WithLastName sets the last name
func (b *ContactBuilder) WithLastName(lastName string) *ContactBuilder {
	b.Set("LastName", lastName)
	return b
}

// WithEmail sets the email
func (b *ContactBuilder) WithEmail(email string) *ContactBuilder {
	b.Set("Email", email)
	return b
}

// WithPhone sets the phone
func (b *ContactBuilder) WithPhone(phone string) *ContactBuilder {
	b.Set("Phone", phone)
	return b
}

// WithAccountId sets the account ID
func (b *ContactBuilder) WithAccountId(accountId string) *ContactBuilder {
	b.Set("AccountId", accountId)
	return b
}

// WithTitle sets the title
func (b *ContactBuilder) WithTitle(title string) *ContactBuilder {
	b.Set("Title", title)
	return b
}

// WithDepartment sets the department
func (b *ContactBuilder) WithDepartment(department string) *ContactBuilder {
	b.Set("Department", department)
	return b
}

// LeadBuilder helps build Lead records
type LeadBuilder struct {
	*RecordBuilder
}

// NewLeadBuilder creates a new Lead builder
func NewLeadBuilder() *LeadBuilder {
	return &LeadBuilder{
		RecordBuilder: NewRecordBuilder("Lead"),
	}
}

// WithFirstName sets the first name
func (b *LeadBuilder) WithFirstName(firstName string) *LeadBuilder {
	b.Set("FirstName", firstName)
	return b
}

// WithLastName sets the last name
func (b *LeadBuilder) WithLastName(lastName string) *LeadBuilder {
	b.Set("LastName", lastName)
	return b
}

// WithCompany sets the company
func (b *LeadBuilder) WithCompany(company string) *LeadBuilder {
	b.Set("Company", company)
	return b
}

// WithEmail sets the email
func (b *LeadBuilder) WithEmail(email string) *LeadBuilder {
	b.Set("Email", email)
	return b
}

// WithPhone sets the phone
func (b *LeadBuilder) WithPhone(phone string) *LeadBuilder {
	b.Set("Phone", phone)
	return b
}

// WithStatus sets the status
func (b *LeadBuilder) WithStatus(status string) *LeadBuilder {
	b.Set("Status", status)
	return b
}

// OpportunityBuilder helps build Opportunity records
type OpportunityBuilder struct {
	*RecordBuilder
}

// NewOpportunityBuilder creates a new Opportunity builder
func NewOpportunityBuilder() *OpportunityBuilder {
	return &OpportunityBuilder{
		RecordBuilder: NewRecordBuilder("Opportunity"),
	}
}

// WithName sets the opportunity name
func (b *OpportunityBuilder) WithName(name string) *OpportunityBuilder {
	b.Set("Name", name)
	return b
}

// WithAccountId sets the account ID
func (b *OpportunityBuilder) WithAccountId(accountId string) *OpportunityBuilder {
	b.Set("AccountId", accountId)
	return b
}

// WithAmount sets the amount
func (b *OpportunityBuilder) WithAmount(amount float64) *OpportunityBuilder {
	b.Set("Amount", amount)
	return b
}

// WithCloseDate sets the close date
func (b *OpportunityBuilder) WithCloseDate(closeDate string) *OpportunityBuilder {
	b.Set("CloseDate", closeDate)
	return b
}

// WithStageName sets the stage
func (b *OpportunityBuilder) WithStageName(stage string) *OpportunityBuilder {
	b.Set("StageName", stage)
	return b
}

// WithProbability sets the probability
func (b *OpportunityBuilder) WithProbability(probability float64) *OpportunityBuilder {
	b.Set("Probability", probability)
	return b
}

// CaseBuilder helps build Case records
type CaseBuilder struct {
	*RecordBuilder
}

// NewCaseBuilder creates a new Case builder
func NewCaseBuilder() *CaseBuilder {
	return &CaseBuilder{
		RecordBuilder: NewRecordBuilder("Case"),
	}
}

// WithSubject sets the subject
func (b *CaseBuilder) WithSubject(subject string) *CaseBuilder {
	b.Set("Subject", subject)
	return b
}

// WithDescription sets the description
func (b *CaseBuilder) WithDescription(description string) *CaseBuilder {
	b.Set("Description", description)
	return b
}

// WithStatus sets the status
func (b *CaseBuilder) WithStatus(status string) *CaseBuilder {
	b.Set("Status", status)
	return b
}

// WithPriority sets the priority
func (b *CaseBuilder) WithPriority(priority string) *CaseBuilder {
	b.Set("Priority", priority)
	return b
}

// WithOrigin sets the origin
func (b *CaseBuilder) WithOrigin(origin string) *CaseBuilder {
	b.Set("Origin", origin)
	return b
}

// WithAccountId sets the account ID
func (b *CaseBuilder) WithAccountId(accountId string) *CaseBuilder {
	b.Set("AccountId", accountId)
	return b
}

// WithContactId sets the contact ID
func (b *CaseBuilder) WithContactId(contactId string) *CaseBuilder {
	b.Set("ContactId", contactId)
	return b
}
