package oidc

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	"github.com/gorilla/sessions"
	"github.com/satori/go.uuid"
)

// Paypal is not fully OIDC compliant, therefor it's not possible to verify the id_token HMAC
// See: https://groups.google.com/forum/#!topic/mod_auth_openidc/fPc_C8rb9ns
type Paypal struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	SessionStore sessions.Store
}

// NewAuth initializes a new OpenID Connect Session and redirects the user
func (o *Paypal) NewAuth(w http.ResponseWriter, r *http.Request) {
	nonce := uuid.NewV4().String()
	state := uuid.NewV4().String()

	vals := url.Values{}
	vals.Add("client_id", o.ClientID)
	vals.Add("response_type", "code")
	vals.Add("scope", "openid")
	vals.Add("redirect_uri", o.RedirectURI)
	vals.Add("nonce", nonce)
	vals.Add("state", state)
	urlParams := vals.Encode()

	// CSRF Prevention using nonce and state
	session, _ := o.SessionStore.Get(r, "poidc")
	session.Values["nonce"] = nonce
	session.Values["state"] = state
	session.Save(r, w)

	http.Redirect(w, r, "https://www.sandbox.paypal.com/webapps/auth/protocol/openidconnect/v1/authorize?"+urlParams, http.StatusFound)
}

// Callback handles the callback from the user after the identity provider provided a code to the users agent
func (o *Paypal) Callback(w http.ResponseWriter, r *http.Request) (uid *string, err error) {
	// Delete CSRF Tokens afterwards
	defer func() {
		session, _ := o.SessionStore.Get(r, "poidc")
		delete(session.Values, "nonce")
		delete(session.Values, "state")
		session.Save(r, w)
	}()
	session, _ := o.SessionStore.Get(r, "poidc")
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
	vals.Add("grant_type", "authorization_code")
	vals.Add("code", code)
	vals.Add("redirect_uri", o.RedirectURI)
	c := http.Client{}
	req, err := http.NewRequest("POST", "https://api.sandbox.paypal.com/v1/identity/openidconnect/tokenservice", strings.NewReader(vals.Encode()))
	req.SetBasicAuth(o.ClientID, o.ClientSecret)
	if err != nil {
		return nil, fmt.Errorf("Could not build request: %s", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := c.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error on token exchange request: %s", err)
	}
	var respValues map[string]string
	dec := json.NewDecoder(resp.Body)
	err = dec.Decode(&respValues)
	if err != nil {
		return nil, fmt.Errorf("Error decoding token exchange resp to json: %s", err)
	}
	// JWT
	token, err := jwt.Parse(respValues["id_token"], func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(o.ClientID), nil
	})
	// PayPal is not compliant and it's not possible to verify the hmac, therefor the Parsing will partly fail
	if token == nil {
		return nil, fmt.Errorf("Could not decode token")
	}

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

	// Request UserInfo endpoint
	uinfoReq, err := http.NewRequest("GET", "https://api.sandbox.paypal.com/v1/identity/openidconnect/userinfo/?schema=openid", nil)
	uinfoReq.Header.Set("Content-Type", "application/json")
	uinfoReq.Header.Set("Authorization", respValues["token_type"]+" "+respValues["access_token"])

	resp, err = c.Do(uinfoReq)
	if err != nil {
		return nil, fmt.Errorf("Error requesting UserInfo endpoint: %s", err)
	}
	var respUserInfo map[string]string
	dec = json.NewDecoder(resp.Body)
	err = dec.Decode(&respUserInfo)
	if err != nil {
		return nil, fmt.Errorf("Could not decode userinfo response: %s", err)
	}

	userID, ok := respUserInfo["user_id"]
	if !ok || userID == "" {
		return nil, fmt.Errorf("Could not find unique user identifier")
	}
	return &userID, nil
}
