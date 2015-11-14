package controller

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-oidc/oidc"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

type AuthController struct {
	OAuthClient *oidc.Client
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
		email, ok := claims["email"]
		if !ok {
			http.Error(w, "Could not find verified email address", http.StatusBadRequest)
			return
		}

		//if name, ok := claims["name"]; !ok {
		//	http.Error(w, "Could not find name", http.StatusBadRequest)
		//	return
		//}

		session := ctx.Value("session").(*sessions.Session)
		session.Values["user"] = email
		session.Save(r, w)

		http.Redirect(w, r, successURL, http.StatusFound)
	})
}
