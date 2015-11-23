package controller

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"posty/model"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-oidc/oidc"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

type AuthDataProvider interface {
	GetByOAuthID(oauthid string) (*model.User, error)
	UpdateLastLogin(id string) error
	NewUser() *model.User
	SaveNew(u *model.User) error
}

type AuthController struct {
	OAuthClient *oidc.Client
	Data        AuthDataProvider
}

func (c *AuthController) InitOAuth(clientID, clientSecret, discovery, oauthRedirectURL string) error {

	// OpenID
	cc := oidc.ClientCredentials{
		ID:     clientID,
		Secret: clientSecret,
	}

	log.Printf("fetching provider config from %s...", discovery)

	var cfg oidc.ProviderConfig
	var err error
	for {
		cfg, err = oidc.FetchProviderConfig(http.DefaultClient, discovery)
		if err == nil {
			break
		}

		log.Printf("failed fetching provider config, trying again: %v", err)
		time.Sleep(3 * time.Second)
	}

	log.Printf("fetched provider config from %s", discovery)

	ccfg := oidc.ClientConfig{
		ProviderConfig: cfg,
		Credentials:    cc,
		RedirectURL:    oauthRedirectURL,
	}

	client, err := oidc.NewClient(ccfg)
	if err != nil {
		log.Fatalf("unable to create Client: %v", err)
		return err
	}

	client.SyncProviderConfig(discovery)
	c.OAuthClient = client
	return nil
}

func (c *AuthController) Login() xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Login")
		oac, err := c.OAuthClient.OAuthClient()
		if err != nil {
			http.Error(w, "oauth failed", http.StatusInternalServerError)
			return
		}

		u, err := url.Parse(oac.AuthCodeURL("", "", ""))
		if err != nil {
			http.Error(w, "oauth failed", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, u.String(), http.StatusFound)
	})
}

func (c *AuthController) Logout(loginURL string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Logout")
		session := ctx.Value("session").(*sessions.Session)
		delete(session.Values, "user")
		session.Save(r, w)

		http.Redirect(w, r, loginURL, http.StatusFound)
	})
}

func (c *AuthController) OAuth2Callback(successURL string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: OAuth2Redirect")
		code := r.URL.Query().Get("code")
		if code == "" {
			http.Error(w, "code query param must be set", http.StatusBadRequest)
			return
		}

		tok, err := c.OAuthClient.ExchangeAuthCode(code)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to verify auth code with issuer: %v", err), http.StatusBadRequest)
			return
		}

		claims, err := tok.Claims()
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to construct claims: %v", err), http.StatusBadRequest)
			return
		}
		_, ok := claims["email"]
		if !ok {
			http.Error(w, "Could not find verified email address", http.StatusBadRequest)
			return
		}

		session := ctx.Value("session").(*sessions.Session)
		u, err := c.loginGoogle(claims)
		if err != nil {
			log.Errorf("Could not authenticate user: Error: %s, Claims: %#v", err, claims)
			http.Error(w, "Could not authenticate", http.StatusInternalServerError)
			return
		}
		session.Values["user"] = u.ID
		session.Save(r, w)

		http.Redirect(w, r, successURL, http.StatusFound)
	})
}

func (c *AuthController) loginGoogle(claims map[string]interface{}) (*model.User, error) {
	isub, ok := claims["sub"]
	if !ok {
		return nil, errors.New("Could not find google identifier")
	}
	sub := isub.(string)

	var username string
	name, ok := claims["name"]
	if ok {
		username, _ = name.(string)
	}

	var email string
	gEmail, ok := claims["email"]
	if ok {
		email, _ = gEmail.(string)
	}
	oid := "google:" + sub
	u, err := c.Data.GetByOAuthID(oid)
	if u == nil || err != nil {
		u = c.Data.NewUser()
		u.OAuthID = oid
		u.Username = username
		u.Email = email
		log.Infof("User to create: %#v", u)
		err = c.Data.SaveNew(u)
		if err != nil {
			return nil, fmt.Errorf("Could not create new user: %s", err)
		}
	}
	err = c.Data.UpdateLastLogin(u.ID)
	if err != nil {
		log.Warnf("Could not update last login for user: %#v", u)
	}

	return u, nil
}
