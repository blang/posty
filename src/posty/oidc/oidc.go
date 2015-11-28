package oidc

import "net/http"

// Provider represents an OpenID Connect client
type Provider interface {
	// NewAuth starts a new OIDC authentication and redirects the user to the identity provider
	NewAuth(w http.ResponseWriter, r *http.Request)
	// Callback receives the callback from the identity provider, verifies it and requests user data
	Callback(w http.ResponseWriter, r *http.Request) (user map[string]string, err error)
}
