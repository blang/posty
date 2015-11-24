package oidc

import "net/http"

// Provider represents an OpenID Connect client
type Provider interface {
	NewAuth(w http.ResponseWriter, r *http.Request)
	Callback(w http.ResponseWriter, r *http.Request) (uid *string, err error)
}
