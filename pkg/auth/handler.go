package auth

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	sferrors "github.com/MASA-JAPAN/go-salesforce-emulator/pkg/errors"
)

// Credential represents OAuth2 credentials
type Credential struct {
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
}

// Handler handles OAuth2 authentication
type Handler struct {
	credentials map[string]Credential // keyed by ClientID
	sessions    *SessionManager
	instanceURL string
	userID      string
	orgID       string
}

// NewHandler creates a new auth handler
func NewHandler(instanceURL, userID, orgID string, tokenLifetime time.Duration) *Handler {
	return &Handler{
		credentials: make(map[string]Credential),
		sessions:    NewSessionManager(tokenLifetime),
		instanceURL: instanceURL,
		userID:      userID,
		orgID:       orgID,
	}
}

// AddCredential adds a valid credential
func (h *Handler) AddCredential(cred Credential) {
	h.credentials[cred.ClientID] = cred
}

// SetInstanceURL updates the instance URL
func (h *Handler) SetInstanceURL(url string) {
	h.instanceURL = url
}

// TokenResponse is the OAuth2 token response
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	InstanceURL string `json:"instance_url"`
	ID          string `json:"id"`
	TokenType   string `json:"token_type"`
	IssuedAt    string `json:"issued_at"`
	Signature   string `json:"signature"`
}

// HandleOAuth handles POST /services/oauth2/token
func (h *Handler) HandleOAuth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.respondError(w, "invalid_request", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		h.respondError(w, "invalid_request", "Invalid form data", http.StatusBadRequest)
		return
	}

	grantType := r.FormValue("grant_type")

	switch grantType {
	case "password":
		h.handlePasswordFlow(w, r)
	case "client_credentials":
		h.handleClientCredentialsFlow(w, r)
	default:
		h.respondError(w, sferrors.ErrorCodeUnsupportedGrantType, "Grant type not supported", http.StatusBadRequest)
	}
}

func (h *Handler) handlePasswordFlow(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Validate credentials
	cred, ok := h.credentials[clientID]
	if !ok {
		h.respondError(w, sferrors.ErrorCodeInvalidGrant, "authentication failure", http.StatusBadRequest)
		return
	}

	if cred.ClientSecret != clientSecret || cred.Username != username || cred.Password != password {
		h.respondError(w, sferrors.ErrorCodeInvalidGrant, "authentication failure", http.StatusBadRequest)
		return
	}

	// Create session
	session := h.sessions.CreateSession(h.instanceURL, h.userID, h.orgID)
	h.respondSuccess(w, session)
}

func (h *Handler) handleClientCredentialsFlow(w http.ResponseWriter, r *http.Request) {
	clientID := r.FormValue("client_id")
	clientSecret := r.FormValue("client_secret")

	// Validate credentials
	cred, ok := h.credentials[clientID]
	if !ok {
		h.respondError(w, sferrors.ErrorCodeInvalidGrant, "authentication failure", http.StatusBadRequest)
		return
	}

	if cred.ClientSecret != clientSecret {
		h.respondError(w, sferrors.ErrorCodeInvalidGrant, "authentication failure", http.StatusBadRequest)
		return
	}

	// Create session
	session := h.sessions.CreateSession(h.instanceURL, h.userID, h.orgID)
	h.respondSuccess(w, session)
}

func (h *Handler) respondSuccess(w http.ResponseWriter, session *Session) {
	response := TokenResponse{
		AccessToken: session.AccessToken,
		InstanceURL: session.InstanceURL,
		ID:          h.instanceURL + "/id/00D000000000000AAA/" + h.userID,
		TokenType:   session.TokenType,
		IssuedAt:    formatIssuedAt(session.IssuedAt),
		Signature:   "mock_signature",
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

func (h *Handler) respondError(w http.ResponseWriter, errorCode, description string, status int) {
	response := sferrors.NewOAuthError(errorCode, description)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(response)
}

// ValidateRequest validates the Authorization header
func (h *Handler) ValidateRequest(r *http.Request) (*Session, error) {
	auth := r.Header.Get("Authorization")
	if auth == "" {
		return nil, sferrors.NewInvalidSessionError()
	}

	// Parse Bearer token
	parts := strings.SplitN(auth, " ", 2)
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return nil, sferrors.NewInvalidSessionError()
	}

	token := parts[1]
	session, ok := h.sessions.GetSession(token)
	if !ok {
		return nil, sferrors.NewInvalidSessionError()
	}

	return session, nil
}

// GetSessionManager returns the session manager for direct access
func (h *Handler) GetSessionManager() *SessionManager {
	return h.sessions
}

func formatIssuedAt(t time.Time) string {
	return t.Format("20060102150405") + "000"
}
