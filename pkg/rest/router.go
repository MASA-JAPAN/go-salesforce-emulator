package rest

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// Router handles REST API routing
type Router struct {
	store       storage.Store
	authHandler *auth.Handler
	apiVersion  string
	routes      []route
}

type route struct {
	pattern *regexp.Regexp
	methods []string
	handler func(http.ResponseWriter, *http.Request, []string)
}

// NewRouter creates a new REST API router
func NewRouter(store storage.Store, authHandler *auth.Handler, apiVersion string) *Router {
	r := &Router{
		store:       store,
		authHandler: authHandler,
		apiVersion:  apiVersion,
	}
	r.setupRoutes()
	return r
}

func (r *Router) setupRoutes() {
	version := regexp.QuoteMeta(r.apiVersion)

	r.routes = []route{
		// SObject operations
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/sobjects/?$`),
			methods: []string{"GET"},
			handler: r.handleDescribeGlobal,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/sobjects/([^/]+)/?$`),
			methods: []string{"GET", "POST"},
			handler: r.handleSObject,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/sobjects/([^/]+)/describe/?$`),
			methods: []string{"GET"},
			handler: r.handleDescribeSObject,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/sobjects/([^/]+)/([^/]+)/?$`),
			methods: []string{"GET", "PATCH", "DELETE"},
			handler: r.handleSObjectRecord,
		},
		// Query
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/query/?$`),
			methods: []string{"GET"},
			handler: r.handleQuery,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/query/([^/]+)/?$`),
			methods: []string{"GET"},
			handler: r.handleQueryMore,
		},
		// Composite
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/composite/sobjects/?$`),
			methods: []string{"POST", "PATCH", "DELETE"},
			handler: r.handleCompositeSObjects,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/composite/?$`),
			methods: []string{"POST"},
			handler: r.handleComposite,
		},
		// Limits
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/limits/?$`),
			methods: []string{"GET"},
			handler: r.handleLimits,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/limits/recordCount/?$`),
			methods: []string{"GET"},
			handler: r.handleRecordCount,
		},
		// Tooling API
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/tooling/query/?$`),
			methods: []string{"GET"},
			handler: r.handleToolingQuery,
		},
		{
			pattern: regexp.MustCompile(`^/services/data/v` + version + `/tooling/sobjects/([^/]+)/?$`),
			methods: []string{"GET", "POST"},
			handler: r.handleToolingSObject,
		},
	}
}

// ServeHTTP handles HTTP requests
func (r *Router) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, Sforce-Query-Options")

	if req.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Check authentication for non-OAuth endpoints
	if !strings.HasPrefix(req.URL.Path, "/services/oauth2/") {
		_, err := r.authHandler.ValidateRequest(req)
		if err != nil {
			r.respondError(w, []sferrors.SalesforceError{err.(sferrors.SalesforceError)}, http.StatusUnauthorized)
			return
		}
	}

	// Find matching route
	path := req.URL.Path
	for _, route := range r.routes {
		matches := route.pattern.FindStringSubmatch(path)
		if matches != nil {
			// Check method
			methodAllowed := false
			for _, m := range route.methods {
				if m == req.Method {
					methodAllowed = true
					break
				}
			}

			if !methodAllowed {
				r.respondError(w, []sferrors.SalesforceError{
					sferrors.NewMethodNotAllowedError(req.Method),
				}, http.StatusMethodNotAllowed)
				return
			}

			route.handler(w, req, matches[1:])
			return
		}
	}

	// Not found
	r.respondError(w, []sferrors.SalesforceError{
		{Message: "The requested resource does not exist", ErrorCode: sferrors.ErrorCodeNotFound},
	}, http.StatusNotFound)
}

func (r *Router) respondError(w http.ResponseWriter, errors []sferrors.SalesforceError, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(errors)
}

func (r *Router) respondJSON(w http.ResponseWriter, data interface{}, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}
