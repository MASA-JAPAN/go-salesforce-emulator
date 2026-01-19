# go-salesforce-emulator

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/MASA-JAPAN/go-salesforce-emulator)](https://goreportcard.com/report/github.com/MASA-JAPAN/go-salesforce-emulator)

A comprehensive Salesforce API emulator for testing Go applications without connecting to a real Salesforce org.

## Features

- **OAuth2 Authentication** - Password and Client Credentials flows
- **SObject CRUD** - Create, Read, Update, Delete operations
- **SOQL Queries** - SELECT, FROM, WHERE, ORDER BY, LIMIT, OFFSET with pagination
- **Bulk Query API** - Job lifecycle with CSV results and Sforce-Locator pagination
- **Composite API** - Batch create/update/delete operations
- **Tooling API** - Query endpoint
- **Metadata API** - SOAP deploy/retrieve operations
- **Limits API** - Limits and RecordCount endpoints
- **Describe** - SObject and Global describe endpoints

### Supported Standard Objects

Account, Contact, Lead, Opportunity, Case, User, Task, Event

## Installation

```bash
go get github.com/MASA-JAPAN/go-salesforce-emulator
```

## Quick Start

### As a Library (for testing)

```go
package myapp_test

import (
    "testing"

    sfemulator "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/emulator"
    sfclient "github.com/MASA-JAPAN/go-salesforce-api-client"
)

func TestMyApp(t *testing.T) {
    // Start emulator
    emu := sfemulator.New()
    emu.Start()
    defer emu.Stop()

    // Pre-load test data
    emu.Store().CreateRecord("Account", map[string]interface{}{
        "Name": "Test Account",
    })

    // Create client pointing to emulator
    creds := emu.GetDefaultCredentials()
    client := &sfclient.Client{
        InstanceURL: emu.URL(),
        AccessToken: emu.CreateTestSession(),
    }

    // Test your code
    result, err := client.Query("SELECT Id, Name FROM Account")
    if err != nil {
        t.Fatal(err)
    }
    if result.TotalSize != 1 {
        t.Errorf("expected 1 record, got %d", result.TotalSize)
    }
}
```

### As a Standalone Server

```bash
# Install
go install github.com/MASA-JAPAN/go-salesforce-emulator/cmd/go-salesforce-emulator@latest

# Run
go-salesforce-emulator -port 8080

# Or with custom credentials
go-salesforce-emulator -port 8080 \
    -client-id myapp \
    -client-secret mysecret \
    -username test@example.com \
    -password password123
```

Then configure your Salesforce client to point to `http://localhost:8080`.

## API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/services/oauth2/token` | POST | OAuth2 token endpoint |
| `/services/data/v58.0/sobjects/{type}` | POST | Create record |
| `/services/data/v58.0/sobjects/{type}/{id}` | GET/PATCH/DELETE | Read/Update/Delete record |
| `/services/data/v58.0/sobjects/{type}/describe` | GET | Describe SObject |
| `/services/data/v58.0/sobjects` | GET | Describe Global |
| `/services/data/v58.0/query` | GET | Execute SOQL query |
| `/services/data/v58.0/composite/sobjects` | POST/PATCH/DELETE | Composite operations |
| `/services/data/v58.0/jobs/query` | POST/GET | Bulk query jobs |
| `/services/data/v58.0/jobs/query/{id}` | GET/PATCH/DELETE | Manage bulk job |
| `/services/data/v58.0/jobs/query/{id}/results` | GET | Get bulk job results |
| `/services/data/v58.0/tooling/query` | GET | Tooling API query |
| `/services/data/v58.0/limits` | GET | API limits |
| `/services/data/v58.0/limits/recordCount` | GET | Record counts |
| `/services/Soap/m/58.0` | POST | Metadata API (SOAP) |

## Configuration Options

```go
emu := sfemulator.New(
    sfemulator.WithAPIVersion("58.0"),
    sfemulator.WithCredentials(sfemulator.Credential{
        ClientID:     "my_client_id",
        ClientSecret: "my_client_secret",
        Username:     "test@example.com",
        Password:     "password123",
    }),
)
```

## Test Utilities

The package includes builders for creating test data:

```go
import "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/testutil"

// Create test account
account := testutil.NewAccountBuilder().
    WithName("Acme Corp").
    WithIndustry("Technology").
    Build()

emu.Store().CreateRecord("Account", account)

// Create test contact
contact := testutil.NewContactBuilder().
    WithFirstName("John").
    WithLastName("Doe").
    WithEmail("john@example.com").
    Build()

emu.Store().CreateRecord("Contact", contact)
```

## Fixtures

Pre-built scenarios for common testing needs:

```go
import "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/testutil"

fixtures := testutil.NewFixtures(emu.Store())

// Load basic CRM data (accounts, contacts, opportunities)
fixtures.LoadBasicCRM()

// Load high-volume test data
fixtures.LoadHighVolume(1000) // Creates 1000 accounts
```

## Compatibility

Designed to work with [go-salesforce-api-client](https://github.com/MASA-JAPAN/go-salesforce-api-client) and other Salesforce REST API clients.

## Contributing

Contributions are welcome! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
