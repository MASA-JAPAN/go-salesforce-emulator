package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/testutil"
)

func main() {
	// Create a new emulator with custom credentials
	emu := emulator.New(
		emulator.WithCredentials(auth.Credential{
			ClientID:     "my_client_id",
			ClientSecret: "my_client_secret",
			Username:     "admin@example.com",
			Password:     "password123",
		}),
	)

	// Start the emulator
	baseURL := emu.Start()
	defer emu.Stop()

	fmt.Printf("Emulator running at: %s\n\n", baseURL)

	client := emu.HTTPClient()

	// Step 1: Authenticate
	token, err := authenticate(client, baseURL, "my_client_id", "my_client_secret", "admin@example.com", "password123")
	if err != nil {
		fmt.Printf("Authentication failed: %v\n", err)
		return
	}
	fmt.Printf("✓ Authenticated successfully\n")
	fmt.Printf("  Access Token: %s...\n\n", token[:20])

	// Step 2: Create an Account
	accountID, err := createAccount(client, baseURL, token, "Acme Corporation", "Technology")
	if err != nil {
		fmt.Printf("Failed to create account: %v\n", err)
		return
	}
	fmt.Printf("✓ Created Account: %s\n\n", accountID)

	// Step 3: Get the Account
	account, err := getAccount(client, baseURL, token, accountID)
	if err != nil {
		fmt.Printf("Failed to get account: %v\n", err)
		return
	}
	fmt.Printf("✓ Retrieved Account:\n")
	fmt.Printf("  ID: %s\n", account["Id"])
	fmt.Printf("  Name: %s\n", account["Name"])
	fmt.Printf("  Industry: %s\n\n", account["Industry"])

	// Step 4: Query accounts
	results, err := queryAccounts(client, baseURL, token)
	if err != nil {
		fmt.Printf("Failed to query accounts: %v\n", err)
		return
	}
	fmt.Printf("✓ Query returned %d record(s)\n\n", results["totalSize"])

	// Step 5: Update the Account
	err = updateAccount(client, baseURL, token, accountID, "Acme Corp International")
	if err != nil {
		fmt.Printf("Failed to update account: %v\n", err)
		return
	}
	fmt.Printf("✓ Updated Account name\n\n")

	// Step 6: Load fixtures and query
	fixtures := testutil.NewFixtures(emu.Store())
	fixtures.LoadSampleAccounts(5)
	fmt.Printf("✓ Loaded 5 sample accounts\n\n")

	results, err = queryAccounts(client, baseURL, token)
	if err != nil {
		fmt.Printf("Failed to query accounts: %v\n", err)
		return
	}
	fmt.Printf("✓ Query now returns %d record(s)\n\n", results["totalSize"])

	// Step 7: Delete the original Account
	err = deleteAccount(client, baseURL, token, accountID)
	if err != nil {
		fmt.Printf("Failed to delete account: %v\n", err)
		return
	}
	fmt.Printf("✓ Deleted Account: %s\n\n", accountID)

	fmt.Println("All operations completed successfully!")
}

func authenticate(client *http.Client, baseURL, clientID, clientSecret, username, password string) (string, error) {
	data := url.Values{
		"grant_type":    {"password"},
		"client_id":     {clientID},
		"client_secret": {clientSecret},
		"username":      {username},
		"password":      {password},
	}

	resp, err := client.PostForm(baseURL+"/services/oauth2/token", data)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	token, ok := result["access_token"].(string)
	if !ok {
		return "", fmt.Errorf("no access token in response")
	}

	return token, nil
}

func createAccount(client *http.Client, baseURL, token, name, industry string) (string, error) {
	data := map[string]interface{}{
		"Name":     name,
		"Industry": industry,
	}

	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("POST", baseURL+"/services/data/v58.0/sobjects/Account", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	id, ok := result["id"].(string)
	if !ok {
		return "", fmt.Errorf("no id in response")
	}

	return id, nil
}

func getAccount(client *http.Client, baseURL, token, accountID string) (map[string]interface{}, error) {
	req, _ := http.NewRequest("GET", baseURL+"/services/data/v58.0/sobjects/Account/"+accountID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func queryAccounts(client *http.Client, baseURL, token string) (map[string]interface{}, error) {
	query := url.QueryEscape("SELECT Id, Name, Industry FROM Account")
	req, _ := http.NewRequest("GET", baseURL+"/services/data/v58.0/query?q="+query, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func updateAccount(client *http.Client, baseURL, token, accountID, newName string) error {
	data := map[string]interface{}{
		"Name": newName,
	}

	body, _ := json.Marshal(data)
	req, _ := http.NewRequest("PATCH", baseURL+"/services/data/v58.0/sobjects/Account/"+accountID, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update failed: %s", string(body))
	}

	return nil
}

func deleteAccount(client *http.Client, baseURL, token, accountID string) error {
	req, _ := http.NewRequest("DELETE", baseURL+"/services/data/v58.0/sobjects/Account/"+accountID, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete failed: %s", string(body))
	}

	return nil
}
