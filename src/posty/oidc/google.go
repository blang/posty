package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	uuid "github.com/satori/go.uuid"
)

// Google represents an OpenID Connect client for http://accounts.google.com
type Google struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	SessionStore sessions.Store
}

// NewAuth initializes a new OpenID Connect Session and redirects the user
func (o *Google) NewAuth(w http.ResponseWriter, r *http.Request) {
	nonce := uuid.NewV4().String()
	state := uuid.NewV4().String()

	vals := url.Values{}
	vals.Add("client_id", o.ClientID)
	vals.Add("response_type", "code")
	// Profile does allow to get the name of the user, no need to request userinfo endpoint
	vals.Add("scope", "openid profile")
	vals.Add("redirect_uri", o.RedirectURI)
	vals.Add("nonce", nonce)
	vals.Add("state", state)
	urlParams := vals.Encode()

	// CSRF Prevention using nonce and state
	session, _ := o.SessionStore.Get(r, "goidc")
	session.Values["nonce"] = nonce
	session.Values["state"] = state
	session.Save(r, w)

	http.Redirect(w, r, "https://accounts.google.com/o/oauth2/auth?"+urlParams, http.StatusFound)
}

// Callback handles the callback from the user after the identity provider provided a code to the users agent
func (o *Google) Callback(w http.ResponseWriter, r *http.Request) (user map[string]string, err error) {
	// Delete CSRF Tokens afterwards
	defer func() {
		session, _ := o.SessionStore.Get(r, "goidc")
		delete(session.Values, "nonce")
		delete(session.Values, "state")
		session.Save(r, w)
	}()
	session, _ := o.SessionStore.Get(r, "goidc")
	oidcState, ok := session.Values["state"].(string)
	if !ok {
		return nil, fmt.Errorf("Session 'state' not found")
	}
	oidcNonce, ok := session.Values["nonce"].(string)
	if !ok {
		return nil, fmt.Errorf("Session 'nonce' not found")
	}

	err = r.ParseForm()
	if err != nil {
		return nil, fmt.Errorf("Could not parse form: %s", err)
	}
	code := r.Form.Get("code")
	if code == "" {
		return nil, fmt.Errorf("Did not receive code")
	}

	// CSRF Prevention using state
	if state := r.Form.Get("state"); state != oidcState {
		return nil, fmt.Errorf("Could not verify CSRF Token 'state': want: %s, got %s", oidcState, state)
	}
	vals := url.Values{}
	vals.Add("code", code)
	vals.Add("redirect_uri", o.RedirectURI)
	vals.Add("client_id", o.ClientID)
	vals.Add("client_secret", o.ClientSecret)
	vals.Add("grant_type", "authorization_code")
	c := http.Client{}

	// Exchange code for token
	req, err := http.NewRequest("POST", "https://www.googleapis.com/oauth2/v4/token", strings.NewReader(vals.Encode()))
	if err != nil {
		return nil, fmt.Errorf("Could not build request: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error on token exchange request: %s", err)
	}
	var respValues map[string]interface{}
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respValues)
	if err != nil {
		return nil, fmt.Errorf("Error decoding token exchange resp to json: %s", err)
	}
	if _, ok := respValues["error"]; ok {
		return nil, fmt.Errorf("Error returned by the api: %v", respValues)
	}
	var idToken string
	idToken, _ = (respValues["id_token"]).(string)
	if idToken == "" {
		return nil, fmt.Errorf("No id token received: %#v", respValues)
	}

	// Check JWT using google certificates
	keyResp, err := http.Get("https://www.googleapis.com/oauth2/v1/certs")
	if err != nil {
		return nil, fmt.Errorf("Could not get keys from server: %s", err)
	}
	keyDec := json.NewDecoder(keyResp.Body)
	defer keyResp.Body.Close()
	var googleKeys map[string]string
	err = keyDec.Decode(&googleKeys)
	if err != nil {
		return nil, fmt.Errorf("Error decoding certificates from json: %s", err)
	}
	keylookup := func(kid string) (string, error) {
		for key, val := range googleKeys {
			if key == kid {
				return val, nil
			}
		}
		return "", fmt.Errorf("Could not find public key for kid: %s", kid)
	}

	// JWT - Verification including signing method
	token, err := jwt.Parse(idToken, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		kid, ok := (token.Header["kid"]).(string)
		if !ok {
			return nil, fmt.Errorf("Key id not found")
		}
		key, err := keylookup(kid)
		if err != nil {
			return nil, err
		}
		return jwt.ParseRSAPublicKeyFromPEM([]byte(key))
	})

	// JWT Successfull
	if token.Valid {

		// CSRF Prevention using nonce
		nonce, ok := (token.Claims["nonce"]).(string)
		if !ok {
			return nil, fmt.Errorf("No nonce in claims: %v", token.Claims)
		}

		if nonce != oidcNonce {
			return nil, fmt.Errorf("Could not verify CSRF Token 'nonce': want: %s, got %s", oidcNonce, nonce)
		}
		// Verify token audience
		aud, ok := (token.Claims["aud"]).(string)
		if !ok {
			return nil, fmt.Errorf("No aud in claims: %v", token.Claims)
		}
		if aud != o.ClientID {
			return nil, fmt.Errorf("Verification of token 'audience' failed: %v", token.Claims)
		}

		user = make(map[string]string)

		// Check 'sub'
		uid, ok := (token.Claims["sub"]).(string)
		if !ok {
			return nil, fmt.Errorf("Could not get a unique user id")
		}
		user["id"] = uid
		name, ok := (token.Claims["name"]).(string)
		if !ok {
			return nil, fmt.Errorf("Could not get the name of the user")
		}
		user["name"] = name
		return user, nil
	} else if ve, ok := err.(*jwt.ValidationError); ok {
		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
			return nil, fmt.Errorf("ID Token is malformed")
		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
			return nil, fmt.Errorf("ID Token is expired or not active yet: %s", err)
		}
		return nil, fmt.Errorf("Could not handle ID Token: %s", err)
	}
	return nil, fmt.Errorf("Could not handle ID Token: %s", err)
}
