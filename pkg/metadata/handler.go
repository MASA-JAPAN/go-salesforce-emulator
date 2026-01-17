package metadata

import (
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/storage"
)

// Handler handles SOAP Metadata API requests
type Handler struct {
	store       storage.Store
	authHandler *auth.Handler
	apiVersion  string
	deployments map[string]*DeploymentStatus
	retrievals  map[string]*RetrievalStatus
	mu          sync.RWMutex
}

// NewHandler creates a new metadata handler
func NewHandler(store storage.Store, authHandler *auth.Handler, apiVersion string) *Handler {
	return &Handler{
		store:       store,
		authHandler: authHandler,
		apiVersion:  apiVersion,
		deployments: make(map[string]*DeploymentStatus),
		retrievals:  make(map[string]*RetrievalStatus),
	}
}

// DeploymentStatus tracks a deployment
type DeploymentStatus struct {
	ID               string
	Status           string
	Done             bool
	Success          bool
	CheckOnly        bool
	NumberTestsTotal int
	NumberTestsCompleted int
	NumberComponentsTotal int
	NumberComponentsDeployed int
	NumberComponentErrors int
	StartDate        time.Time
	CompletedDate    time.Time
	ErrorMessage     string
}

// RetrievalStatus tracks a retrieval
type RetrievalStatus struct {
	ID            string
	Status        string
	Done          bool
	Success       bool
	ZipFile       string
	StartDate     time.Time
	CompletedDate time.Time
	ErrorMessage  string
}

// SOAPEnvelope represents a SOAP envelope
type SOAPEnvelope struct {
	XMLName xml.Name    `xml:"Envelope"`
	Header  *SOAPHeader `xml:"Header"`
	Body    SOAPBody    `xml:"Body"`
}

// SOAPHeader represents the SOAP header
type SOAPHeader struct {
	SessionHeader *SessionHeader `xml:"SessionHeader"`
}

// SessionHeader contains the session ID
type SessionHeader struct {
	SessionID string `xml:"sessionId"`
}

// SOAPBody represents the SOAP body
type SOAPBody struct {
	Deploy              *DeployRequest              `xml:"deploy"`
	CheckDeployStatus   *CheckDeployStatusRequest   `xml:"checkDeployStatus"`
	CancelDeploy        *CancelDeployRequest        `xml:"cancelDeploy"`
	Retrieve            *RetrieveRequest            `xml:"retrieve"`
	CheckRetrieveStatus *CheckRetrieveStatusRequest `xml:"checkRetrieveStatus"`
}

// DeployRequest represents a deploy request
type DeployRequest struct {
	ZipFile        string        `xml:"ZipFile"`
	DeployOptions  DeployOptions `xml:"DeployOptions"`
}

// DeployOptions contains deployment options
type DeployOptions struct {
	CheckOnly       bool   `xml:"checkOnly"`
	RollbackOnError bool   `xml:"rollbackOnError"`
	TestLevel       string `xml:"testLevel"`
}

// CheckDeployStatusRequest represents a check deploy status request
type CheckDeployStatusRequest struct {
	AsyncProcessId string `xml:"asyncProcessId"`
	IncludeDetails bool   `xml:"includeDetails"`
}

// CancelDeployRequest represents a cancel deploy request
type CancelDeployRequest struct {
	AsyncProcessId string `xml:"asyncProcessId"`
}

// RetrieveRequest represents a retrieve request
type RetrieveRequest struct {
	RetrieveRequest RetrieveRequestBody `xml:"retrieveRequest"`
}

// RetrieveRequestBody contains the retrieve request details
type RetrieveRequestBody struct {
	ApiVersion     string   `xml:"apiVersion"`
	PackageNames   []string `xml:"packageNames"`
	SinglePackage  bool     `xml:"singlePackage"`
	Unpackaged     *Package `xml:"unpackaged"`
}

// Package represents a package manifest
type Package struct {
	Types   []PackageTypeMembers `xml:"types"`
	Version string               `xml:"version"`
}

// PackageTypeMembers represents types in a package
type PackageTypeMembers struct {
	Members []string `xml:"members"`
	Name    string   `xml:"name"`
}

// CheckRetrieveStatusRequest represents a check retrieve status request
type CheckRetrieveStatusRequest struct {
	AsyncProcessId string `xml:"asyncProcessId"`
	IncludeZip     bool   `xml:"includeZip"`
}

// HandleSOAP handles POST /services/Soap/m/XX.X
func (h *Handler) HandleSOAP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondSOAPFault(w, "soapenv:Client", "Method not allowed")
		return
	}

	// Read the body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		h.respondSOAPFault(w, "soapenv:Client", "Failed to read request body")
		return
	}

	// Parse SOAP envelope
	var envelope SOAPEnvelope
	if err := xml.Unmarshal(body, &envelope); err != nil {
		h.respondSOAPFault(w, "soapenv:Client", "Invalid SOAP request: "+err.Error())
		return
	}

	// Validate session
	if envelope.Header == nil || envelope.Header.SessionHeader == nil {
		h.respondSOAPFault(w, "sf:INVALID_SESSION_ID", "Session header required")
		return
	}

	sessionID := envelope.Header.SessionHeader.SessionID
	if _, ok := h.authHandler.GetSessionManager().GetSession(sessionID); !ok {
		h.respondSOAPFault(w, "sf:INVALID_SESSION_ID", "Invalid Session ID found in SessionHeader")
		return
	}

	// Route to appropriate handler
	switch {
	case envelope.Body.Deploy != nil:
		h.handleDeploy(w, envelope.Body.Deploy)
	case envelope.Body.CheckDeployStatus != nil:
		h.handleCheckDeployStatus(w, envelope.Body.CheckDeployStatus)
	case envelope.Body.CancelDeploy != nil:
		h.handleCancelDeploy(w, envelope.Body.CancelDeploy)
	case envelope.Body.Retrieve != nil:
		h.handleRetrieve(w, envelope.Body.Retrieve)
	case envelope.Body.CheckRetrieveStatus != nil:
		h.handleCheckRetrieveStatus(w, envelope.Body.CheckRetrieveStatus)
	default:
		h.respondSOAPFault(w, "soapenv:Client", "Unsupported operation")
	}
}

// handleDeploy handles deploy requests
func (h *Handler) handleDeploy(w http.ResponseWriter, req *DeployRequest) {
	// Validate ZIP file
	if req.ZipFile == "" {
		h.respondSOAPFault(w, "sf:INVALID_ZIP", "ZipFile is required")
		return
	}

	// Decode base64 to verify it's valid
	_, err := base64.StdEncoding.DecodeString(req.ZipFile)
	if err != nil {
		h.respondSOAPFault(w, "sf:INVALID_ZIP", "Invalid base64 encoding for ZipFile")
		return
	}

	// Create deployment
	h.mu.Lock()
	deployID := generateID("0Af")
	status := &DeploymentStatus{
		ID:               deployID,
		Status:           "Pending",
		Done:             false,
		Success:          false,
		CheckOnly:        req.DeployOptions.CheckOnly,
		StartDate:        time.Now(),
		NumberComponentsTotal: 1,
	}
	h.deployments[deployID] = status
	h.mu.Unlock()

	// Simulate async deployment
	go h.processDeploy(deployID)

	// Return async result
	h.respondDeployResult(w, deployID, false, "Pending")
}

// processDeploy simulates deployment processing
func (h *Handler) processDeploy(deployID string) {
	h.mu.Lock()
	status := h.deployments[deployID]
	h.mu.Unlock()

	if status == nil {
		return
	}

	// Simulate processing delay
	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	status.Status = "InProgress"
	h.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	status.Status = "Succeeded"
	status.Done = true
	status.Success = true
	status.NumberComponentsDeployed = status.NumberComponentsTotal
	status.CompletedDate = time.Now()
	h.mu.Unlock()
}

// handleCheckDeployStatus handles check deploy status requests
func (h *Handler) handleCheckDeployStatus(w http.ResponseWriter, req *CheckDeployStatusRequest) {
	h.mu.RLock()
	status, ok := h.deployments[req.AsyncProcessId]
	h.mu.RUnlock()

	if !ok {
		h.respondSOAPFault(w, "sf:INVALID_ID", "Deployment not found")
		return
	}

	h.respondCheckDeployStatus(w, status, req.IncludeDetails)
}

// handleCancelDeploy handles cancel deploy requests
func (h *Handler) handleCancelDeploy(w http.ResponseWriter, req *CancelDeployRequest) {
	h.mu.Lock()
	status, ok := h.deployments[req.AsyncProcessId]
	if ok && !status.Done {
		status.Status = "Canceled"
		status.Done = true
		status.Success = false
		status.CompletedDate = time.Now()
	}
	h.mu.Unlock()

	if !ok {
		h.respondSOAPFault(w, "sf:INVALID_ID", "Deployment not found")
		return
	}

	h.respondCancelDeployResult(w, status)
}

// handleRetrieve handles retrieve requests
func (h *Handler) handleRetrieve(w http.ResponseWriter, req *RetrieveRequest) {
	// Create retrieval
	h.mu.Lock()
	retrieveID := generateID("09S")
	status := &RetrievalStatus{
		ID:        retrieveID,
		Status:    "Pending",
		Done:      false,
		Success:   false,
		StartDate: time.Now(),
	}
	h.retrievals[retrieveID] = status
	h.mu.Unlock()

	// Simulate async retrieval
	go h.processRetrieve(retrieveID)

	// Return async result
	h.respondRetrieveResult(w, retrieveID, false, "Pending")
}

// processRetrieve simulates retrieval processing
func (h *Handler) processRetrieve(retrieveID string) {
	h.mu.Lock()
	status := h.retrievals[retrieveID]
	h.mu.Unlock()

	if status == nil {
		return
	}

	time.Sleep(100 * time.Millisecond)

	h.mu.Lock()
	status.Status = "InProgress"
	h.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	// Create a simple ZIP file (just a package.xml)
	zipContent := createMinimalZip()

	h.mu.Lock()
	status.Status = "Succeeded"
	status.Done = true
	status.Success = true
	status.ZipFile = base64.StdEncoding.EncodeToString(zipContent)
	status.CompletedDate = time.Now()
	h.mu.Unlock()
}

// handleCheckRetrieveStatus handles check retrieve status requests
func (h *Handler) handleCheckRetrieveStatus(w http.ResponseWriter, req *CheckRetrieveStatusRequest) {
	h.mu.RLock()
	status, ok := h.retrievals[req.AsyncProcessId]
	h.mu.RUnlock()

	if !ok {
		h.respondSOAPFault(w, "sf:INVALID_ID", "Retrieval not found")
		return
	}

	h.respondCheckRetrieveStatus(w, status, req.IncludeZip)
}

func (h *Handler) respondSOAPFault(w http.ResponseWriter, faultCode, faultString string) {
	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/">
  <soapenv:Body>
    <soapenv:Fault>
      <faultcode>%s</faultcode>
      <faultstring>%s</faultstring>
    </soapenv:Fault>
  </soapenv:Body>
</soapenv:Envelope>`, faultCode, faultString)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusInternalServerError)
	_, _ = w.Write([]byte(response))
}

func (h *Handler) respondDeployResult(w http.ResponseWriter, id string, done bool, status string) {
	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:sf="http://soap.sforce.com/2006/04/metadata">
  <soapenv:Body>
    <deployResponse>
      <result>
        <done>%t</done>
        <id>%s</id>
        <state>%s</state>
      </result>
    </deployResponse>
  </soapenv:Body>
</soapenv:Envelope>`, done, id, status)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (h *Handler) respondCheckDeployStatus(w http.ResponseWriter, status *DeploymentStatus, includeDetails bool) {
	successStr := "false"
	if status.Success {
		successStr = "true"
	}

	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:sf="http://soap.sforce.com/2006/04/metadata">
  <soapenv:Body>
    <checkDeployStatusResponse>
      <result>
        <checkOnly>%t</checkOnly>
        <done>%t</done>
        <id>%s</id>
        <numberComponentErrors>%d</numberComponentErrors>
        <numberComponentsDeployed>%d</numberComponentsDeployed>
        <numberComponentsTotal>%d</numberComponentsTotal>
        <numberTestErrors>0</numberTestErrors>
        <numberTestsCompleted>%d</numberTestsCompleted>
        <numberTestsTotal>%d</numberTestsTotal>
        <status>%s</status>
        <success>%s</success>
      </result>
    </checkDeployStatusResponse>
  </soapenv:Body>
</soapenv:Envelope>`, status.CheckOnly, status.Done, status.ID, status.NumberComponentErrors,
		status.NumberComponentsDeployed, status.NumberComponentsTotal,
		status.NumberTestsCompleted, status.NumberTestsTotal, status.Status, successStr)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (h *Handler) respondCancelDeployResult(w http.ResponseWriter, status *DeploymentStatus) {
	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:sf="http://soap.sforce.com/2006/04/metadata">
  <soapenv:Body>
    <cancelDeployResponse>
      <result>
        <done>%t</done>
        <id>%s</id>
      </result>
    </cancelDeployResponse>
  </soapenv:Body>
</soapenv:Envelope>`, status.Done, status.ID)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (h *Handler) respondRetrieveResult(w http.ResponseWriter, id string, done bool, status string) {
	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:sf="http://soap.sforce.com/2006/04/metadata">
  <soapenv:Body>
    <retrieveResponse>
      <result>
        <done>%t</done>
        <id>%s</id>
        <state>%s</state>
      </result>
    </retrieveResponse>
  </soapenv:Body>
</soapenv:Envelope>`, done, id, status)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func (h *Handler) respondCheckRetrieveStatus(w http.ResponseWriter, status *RetrievalStatus, includeZip bool) {
	successStr := "false"
	if status.Success {
		successStr = "true"
	}

	zipElement := ""
	if includeZip && status.ZipFile != "" {
		zipElement = fmt.Sprintf("<zipFile>%s</zipFile>", status.ZipFile)
	}

	response := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<soapenv:Envelope xmlns:soapenv="http://schemas.xmlsoap.org/soap/envelope/" xmlns:sf="http://soap.sforce.com/2006/04/metadata">
  <soapenv:Body>
    <checkRetrieveStatusResponse>
      <result>
        <done>%t</done>
        <id>%s</id>
        <status>%s</status>
        <success>%s</success>
        %s
      </result>
    </checkRetrieveStatusResponse>
  </soapenv:Body>
</soapenv:Envelope>`, status.Done, status.ID, status.Status, successStr, zipElement)

	w.Header().Set("Content-Type", "text/xml")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(response))
}

func generateID(prefix string) string {
	// Simple ID generation
	return fmt.Sprintf("%s%015d", prefix, time.Now().UnixNano()%1000000000000000)
}

// createMinimalZip creates a minimal ZIP file with just package.xml
func createMinimalZip() []byte {
	// This is a minimal valid ZIP file containing package.xml
	// In a real implementation, this would be a properly constructed ZIP
	packageXML := `<?xml version="1.0" encoding="UTF-8"?>
<Package xmlns="http://soap.sforce.com/2006/04/metadata">
    <version>58.0</version>
</Package>`
	return []byte(packageXML)
}

// RegisterRoutes registers the metadata API routes
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	path := fmt.Sprintf("/services/Soap/m/%s", strings.TrimPrefix(h.apiVersion, "v"))
	mux.HandleFunc(path, h.HandleSOAP)
	mux.HandleFunc(path+"/", h.HandleSOAP)
}
