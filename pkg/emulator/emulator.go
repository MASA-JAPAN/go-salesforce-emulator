package emulator

import (
	"fmt"
	"net/http"
	"net/http/httptest"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/bulk"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/rest"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// Emulator represents a Salesforce API emulator
type Emulator struct {
	server      *httptest.Server
	store       *storage.MemoryStore
	config      *Config
	authHandler *auth.Handler
	restRouter  *rest.Router
	bulkHandler *bulk.Handler
	mux         *http.ServeMux
}

// New creates a new Salesforce emulator with the given options
func New(opts ...Option) *Emulator {
	config := DefaultConfig()
	for _, opt := range opts {
		opt(config)
	}

	store := storage.NewMemoryStore()

	e := &Emulator{
		store:  store,
		config: config,
		mux:    http.NewServeMux(),
	}

	return e
}

// Start starts the emulator server and returns the base URL
func (e *Emulator) Start() string {
	// Create the test server first to get the URL
	e.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e.mux.ServeHTTP(w, r)
	}))

	// Initialize handlers with the server URL
	e.authHandler = auth.NewHandler(
		e.server.URL,
		e.store.GetDefaultUserID(),
		"00D000000000000AAA",
		e.config.TokenLifetime,
	)

	// Add credentials
	for _, cred := range e.config.Credentials {
		e.authHandler.AddCredential(cred)
	}

	// If no credentials configured, add a default one
	if len(e.config.Credentials) == 0 {
		e.authHandler.AddCredential(auth.Credential{
			ClientID:     "test_client_id",
			ClientSecret: "test_client_secret",
			Username:     "test@example.com",
			Password:     "testpassword",
		})
	}

	// Create REST router
	e.restRouter = rest.NewRouter(e.store, e.authHandler, e.config.APIVersion)

	// Create Bulk handler
	e.bulkHandler = bulk.NewHandler(e.store, e.authHandler, e.config.APIVersion)

	// Setup routes
	e.setupRoutes()

	return e.server.URL
}

func (e *Emulator) setupRoutes() {
	// OAuth endpoints
	e.mux.HandleFunc("/services/oauth2/token", e.authHandler.HandleOAuth)

	// Bulk API endpoints
	e.mux.HandleFunc("/services/data/v"+e.config.APIVersion+"/jobs/query", e.bulkHandler.HandleJobs)
	e.mux.HandleFunc("/services/data/v"+e.config.APIVersion+"/jobs/query/", e.bulkHandler.HandleJobByID)

	// All other REST API endpoints
	e.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		e.restRouter.ServeHTTP(w, r)
	})
}

// Stop stops the emulator server
func (e *Emulator) Stop() {
	if e.server != nil {
		e.server.Close()
	}
}

// URL returns the emulator's base URL (instance URL)
func (e *Emulator) URL() string {
	if e.server == nil {
		return ""
	}
	return e.server.URL
}

// Store returns the underlying storage for direct manipulation
func (e *Emulator) Store() *storage.MemoryStore {
	return e.store
}

// Reset clears all data and resets to initial state
func (e *Emulator) Reset() {
	e.store.Reset()
}

// AuthHandler returns the auth handler for creating sessions directly
func (e *Emulator) AuthHandler() *auth.Handler {
	return e.authHandler
}

// CreateTestSession creates a test session and returns the access token
func (e *Emulator) CreateTestSession() string {
	if e.authHandler == nil {
		return ""
	}
	session := e.authHandler.GetSessionManager().CreateSession(
		e.server.URL,
		e.store.GetDefaultUserID(),
		"00D000000000000AAA",
	)
	return session.AccessToken
}

// GetDefaultCredentials returns the default test credentials
func GetDefaultCredentials() (clientID, clientSecret, username, password string) {
	return "test_client_id", "test_client_secret", "test@example.com", "testpassword"
}

// HTTPClient returns an HTTP client configured to work with the emulator
func (e *Emulator) HTTPClient() *http.Client {
	return e.server.Client()
}

// String returns a string representation of the emulator status
func (e *Emulator) String() string {
	if e.server == nil {
		return "Salesforce Emulator (not started)"
	}
	return fmt.Sprintf("Salesforce Emulator running at %s (API v%s)", e.server.URL, e.config.APIVersion)
}
