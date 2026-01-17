package integration_test

import (
	"testing"
	"time"

	sfclient "github.com/MASA-JAPAN/go-salesforce-api-client"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/testutil"
)

// TestAuthenticationPassword tests OAuth2 password flow
func TestAuthenticationPassword(t *testing.T) {
	emu := emulator.New(
		emulator.WithCredentials(auth.Credential{
			ClientID:     "test_client",
			ClientSecret: "test_secret",
			Username:     "test@example.com",
			Password:     "password",
		}),
	)
	baseURL := emu.Start()
	defer emu.Stop()

	authHandler := sfclient.Auth{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		Username:     "test@example.com",
		Password:     "password",
		TokenURL:     baseURL + "/services/oauth2/token",
	}

	client, err := authHandler.AuthenticatePassword()
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	if client.AccessToken == "" {
		t.Error("Access token is empty")
	}

	if client.InstanceURL != baseURL {
		t.Errorf("Instance URL mismatch: expected %s, got %s", baseURL, client.InstanceURL)
	}
}

// TestAuthenticationClientCredentials tests OAuth2 client credentials flow
func TestAuthenticationClientCredentials(t *testing.T) {
	emu := emulator.New(
		emulator.WithCredentials(auth.Credential{
			ClientID:     "test_client",
			ClientSecret: "test_secret",
		}),
	)
	baseURL := emu.Start()
	defer emu.Stop()

	authHandler := sfclient.Auth{
		ClientID:     "test_client",
		ClientSecret: "test_secret",
		TokenURL:     baseURL + "/services/oauth2/token",
	}

	client, err := authHandler.AuthenticateClientCredentials()
	if err != nil {
		t.Fatalf("Authentication failed: %v", err)
	}

	if client.AccessToken == "" {
		t.Error("Access token is empty")
	}
}

// TestCreateRecord tests record creation
func TestCreateRecord(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	record := map[string]interface{}{
		"Name":     "Test Account",
		"Industry": "Technology",
	}

	resp, err := client.CreateRecord("Account", record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	if !resp.Success {
		t.Error("CreateRecord should return success=true")
	}

	if resp.ID == "" {
		t.Error("CreateRecord should return an ID")
	}
}

// TestGetRecord tests record retrieval
func TestGetRecord(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create a record first
	record := map[string]interface{}{
		"Name":     "Test Account",
		"Industry": "Technology",
	}

	createResp, err := client.CreateRecord("Account", record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Get the record
	result, err := client.GetRecord("Account", createResp.ID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if result["Name"] != "Test Account" {
		t.Errorf("Expected Name='Test Account', got %v", result["Name"])
	}

	if result["Industry"] != "Technology" {
		t.Errorf("Expected Industry='Technology', got %v", result["Industry"])
	}
}

// TestUpdateRecord tests record update
func TestUpdateRecord(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create a record first
	record := map[string]interface{}{
		"Name":     "Original Name",
		"Industry": "Technology",
	}

	createResp, err := client.CreateRecord("Account", record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Update the record
	updates := map[string]interface{}{
		"Name": "Updated Name",
	}

	err = client.UpdateRecord("Account", createResp.ID, updates)
	if err != nil {
		t.Fatalf("UpdateRecord failed: %v", err)
	}

	// Verify the update
	result, err := client.GetRecord("Account", createResp.ID)
	if err != nil {
		t.Fatalf("GetRecord failed: %v", err)
	}

	if result["Name"] != "Updated Name" {
		t.Errorf("Expected Name='Updated Name', got %v", result["Name"])
	}
}

// TestDeleteRecord tests record deletion
func TestDeleteRecord(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create a record first
	record := map[string]interface{}{
		"Name": "To Be Deleted",
	}

	createResp, err := client.CreateRecord("Account", record)
	if err != nil {
		t.Fatalf("CreateRecord failed: %v", err)
	}

	// Delete the record
	err = client.DeleteRecord("Account", createResp.ID)
	if err != nil {
		t.Fatalf("DeleteRecord failed: %v", err)
	}

	// Verify the record is deleted (should get an error)
	_, err = client.GetRecord("Account", createResp.ID)
	if err == nil {
		t.Error("Expected error when getting deleted record")
	}
}

// TestQuery tests SOQL queries
func TestQuery(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create some records
	for i := 0; i < 5; i++ {
		record := map[string]interface{}{
			"Name":     "Account " + string(rune('A'+i)),
			"Industry": "Technology",
		}
		_, err := client.CreateRecord("Account", record)
		if err != nil {
			t.Fatalf("CreateRecord failed: %v", err)
		}
	}

	// Query the records
	result, err := client.Query("SELECT Id, Name, Industry FROM Account WHERE Industry = 'Technology'")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if result.TotalSize != 5 {
		t.Errorf("Expected TotalSize=5, got %d", result.TotalSize)
	}

	if len(result.Records) != 5 {
		t.Errorf("Expected 5 records, got %d", len(result.Records))
	}
}

// TestQueryWithPagination tests query pagination
func TestQueryWithPagination(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	// Load high volume test data
	fixtures := testutil.NewFixtures(emu.Store())
	_, err := fixtures.LoadSampleAccounts(100)
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	client := createAuthenticatedClient(t, emu, baseURL)

	// Query with explicit pagination
	result, err := client.Query("SELECT Id, Name FROM Account LIMIT 50")
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if len(result.Records) != 50 {
		t.Errorf("Expected 50 records, got %d", len(result.Records))
	}
}

// TestDescribeSObject tests the describe sobject endpoint
func TestDescribeSObject(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	result, err := client.DescribeSObject("Account")
	if err != nil {
		t.Fatalf("DescribeSObject failed: %v", err)
	}

	if result["name"] != "Account" {
		t.Errorf("Expected name='Account', got %v", result["name"])
	}
}

// TestBulkQuery tests the bulk query API
func TestBulkQuery(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	// Load test data
	fixtures := testutil.NewFixtures(emu.Store())
	_, err := fixtures.LoadSampleAccounts(10)
	if err != nil {
		t.Fatalf("Failed to load fixtures: %v", err)
	}

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create bulk query job
	job, err := client.CreateJobQuery("SELECT Id, Name FROM Account")
	if err != nil {
		t.Fatalf("CreateJobQuery failed: %v", err)
	}

	if job.ID == "" {
		t.Error("Job ID should not be empty")
	}

	// Wait for job to complete
	time.Sleep(500 * time.Millisecond)

	// Check job status
	status, err := client.GetJobQuery(job.ID)
	if err != nil {
		t.Fatalf("GetJobQuery failed: %v", err)
	}

	if status.State != "JobComplete" {
		t.Errorf("Expected state=JobComplete, got %s", status.State)
	}

	// Get results
	results, _, err := client.GetJobQueryResultsParsed(job.ID, "", 100)
	if err != nil {
		t.Fatalf("GetJobQueryResultsParsed failed: %v", err)
	}

	if len(results) != 10 {
		t.Errorf("Expected 10 results, got %d", len(results))
	}
}

// TestGetLimits tests the limits API
func TestGetLimits(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	limits, err := client.GetLimits()
	if err != nil {
		t.Fatalf("GetLimits failed: %v", err)
	}

	if limits == nil {
		t.Error("Limits should not be nil")
	}

	// Check that DailyApiRequests exists
	if limits["DailyApiRequests"] == nil {
		t.Error("DailyApiRequests should be present")
	}
}

// TestGetRecordCounts tests the record count API
func TestGetRecordCounts(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	// Load test data
	fixtures := testutil.NewFixtures(emu.Store())
	fixtures.LoadSampleAccounts(5)
	fixtures.LoadSampleContacts(10, nil)

	client := createAuthenticatedClient(t, emu, baseURL)

	counts, err := client.GetRecordCounts([]string{"Account", "Contact"})
	if err != nil {
		t.Fatalf("GetRecordCounts failed: %v", err)
	}

	if counts == nil {
		t.Error("Counts should not be nil")
	}
}

// TestCompositeCreate tests composite create operations
func TestCompositeCreate(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	records := []map[string]interface{}{
		{"Name": "Account 1"},
		{"Name": "Account 2"},
	}

	results, err := client.CreateRecords("Account", records)
	if err != nil {
		t.Fatalf("CreateRecords failed: %v", err)
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	for i, result := range results {
		if !result.Success {
			t.Errorf("Record %d should be successful", i)
		}
		if result.ID == "" {
			t.Errorf("Record %d should have an ID", i)
		}
	}
}

// TestCompositeUpdate tests composite update operations
func TestCompositeUpdate(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create records first
	createRecords := []map[string]interface{}{
		{"Name": "Original 1"},
		{"Name": "Original 2"},
	}

	createResults, err := client.CreateRecords("Account", createRecords)
	if err != nil {
		t.Fatalf("CreateRecords failed: %v", err)
	}

	// Update records
	updateRecords := []map[string]interface{}{
		{
			"Id":   createResults[0].ID,
			"Name": "Updated 1",
		},
		{
			"Id":   createResults[1].ID,
			"Name": "Updated 2",
		},
	}

	err = client.UpdateRecords("Account", updateRecords)
	if err != nil {
		t.Fatalf("UpdateRecords failed: %v", err)
	}

	// Verify updates
	result1, _ := client.GetRecord("Account", createResults[0].ID)
	if result1["Name"] != "Updated 1" {
		t.Errorf("Expected Name='Updated 1', got %v", result1["Name"])
	}
}

// TestCompositeDelete tests composite delete operations
func TestCompositeDelete(t *testing.T) {
	emu := emulator.New()
	baseURL := emu.Start()
	defer emu.Stop()

	client := createAuthenticatedClient(t, emu, baseURL)

	// Create records first
	createRecords := []map[string]interface{}{
		{"Name": "To Delete 1"},
		{"Name": "To Delete 2"},
	}

	createResults, err := client.CreateRecords("Account", createRecords)
	if err != nil {
		t.Fatalf("CreateRecords failed: %v", err)
	}

	ids := []string{createResults[0].ID, createResults[1].ID}

	// Delete records
	err = client.DeleteRecords("Account", ids)
	if err != nil {
		t.Fatalf("DeleteRecords failed: %v", err)
	}

	// Verify deletion
	_, err = client.GetRecord("Account", createResults[0].ID)
	if err == nil {
		t.Error("Expected error when getting deleted record")
	}
}

// Helper function to create an authenticated client
func createAuthenticatedClient(t *testing.T, emu *emulator.Emulator, baseURL string) *sfclient.Client {
	clientID, clientSecret, username, password := emulator.GetDefaultCredentials()

	authHandler := sfclient.Auth{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Username:     username,
		Password:     password,
		TokenURL:     baseURL + "/services/oauth2/token",
	}

	client, err := authHandler.AuthenticatePassword()
	if err != nil {
		t.Fatalf("Failed to authenticate: %v", err)
	}

	return client
}
