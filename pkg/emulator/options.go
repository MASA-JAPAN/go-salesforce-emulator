package emulator

import (
	"time"

	"github.com/MASA-JAPAN/go-salesforce-emulator/pkg/auth"
)

// Config holds the emulator configuration
type Config struct {
	// APIVersion is the Salesforce API version to emulate (default: "58.0")
	APIVersion string

	// Credentials are the valid OAuth credentials
	Credentials []auth.Credential

	// TokenLifetime is how long tokens are valid (default: 2 hours)
	TokenLifetime time.Duration

	// Port is the port to listen on (0 for random)
	Port int
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		APIVersion:    "58.0",
		Credentials:   []auth.Credential{},
		TokenLifetime: 2 * time.Hour,
		Port:          0,
	}
}

// Option is a function that modifies the config
type Option func(*Config)

// WithAPIVersion sets the API version
func WithAPIVersion(version string) Option {
	return func(c *Config) {
		c.APIVersion = version
	}
}

// WithCredentials adds valid OAuth credentials
func WithCredentials(creds ...auth.Credential) Option {
	return func(c *Config) {
		c.Credentials = append(c.Credentials, creds...)
	}
}

// WithCredential adds a single valid OAuth credential
func WithCredential(clientID, clientSecret, username, password string) Option {
	return func(c *Config) {
		c.Credentials = append(c.Credentials, auth.Credential{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			Username:     username,
			Password:     password,
		})
	}
}

// WithTokenLifetime sets the token lifetime
func WithTokenLifetime(d time.Duration) Option {
	return func(c *Config) {
		c.TokenLifetime = d
	}
}

// WithPort sets the port to listen on
func WithPort(port int) Option {
	return func(c *Config) {
		c.Port = port
	}
}
