package main

import (
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/coreos/go-oidc/oidc"
	gctx "github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

var (
	listen           = flag.String("http", ":8080", "Listen on")
	clientID         = flag.String("client-id", "", "OAuth client ID")
	clientSecret     = flag.String("client-secret", "", "OAuth client secret")
	oauthRedirectURL = flag.String("oauth-redirect-url", "http://127.0.0.1:8080/oauth2cb", "http://[host]/oauth2cb")
	sessionHashKey   = flag.String("session-hash-key", "", "Session hash key, 32/64 Byte")
	sessionBlockKey  = flag.String("session-block-key", "", "Session block encryption key, valid lengths are 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256")
)

const (
	oauthDiscovery = "https://accounts.google.com"
)

func checkFlags() bool {
	flag.Parse()
	if *listen == "" {
		log.Fatal("Flag 'listen' must be set")
		return false
	}
	if *clientID == "" {
		log.Fatal("Flag 'client-id' must be set")
		return false
	}
	if *clientSecret == "" {
		log.Fatal("Flag 'client-secret' must be set")
		return false
	}
	if *oauthRedirectURL == "" {
		log.Fatal("Flag 'oauth-redirect-url' must be set")
		return false
	}
	if *sessionHashKey == "" {
		*sessionHashKey = string(securecookie.GenerateRandomKey(64))
	}
	if *sessionBlockKey == "" {
		*sessionBlockKey = string(securecookie.GenerateRandomKey(32))
	}
	return true
}

func main() {
	if !checkFlags() {
		os.Exit(1)
	}
	// OpenID
	cc := oidc.ClientCredentials{
		ID:     *clientID,
		Secret: *clientSecret,
	}

	log.Printf("fetching provider config from %s...", oauthDiscovery)

	var cfg oidc.ProviderConfig
	var err error
	for {
		cfg, err = oidc.FetchProviderConfig(http.DefaultClient, oauthDiscovery)
		if err == nil {
			break
		}

		log.Printf("failed fetching provider config, trying again: %v", err)
		time.Sleep(3 * time.Second)
	}

	log.Printf("fetched provider config from %s", oauthDiscovery)

	ccfg := oidc.ClientConfig{
		ProviderConfig: cfg,
		Credentials:    cc,
		RedirectURL:    *oauthRedirectURL,
	}

	client, err := oidc.NewClient(ccfg)
	if err != nil {
		log.Fatalf("unable to create Client: %v", err)
	}

	client.SyncProviderConfig(oauthDiscovery)

	// Middleware
	baseChain := xhandler.Chain{}
	baseChain.UseC(xhandler.TimeoutHandler(2 * time.Second))

	// Session management
	sessionStore := sessions.NewCookieStore([]byte(*sessionHashKey), []byte(*sessionBlockKey))
	baseChain.UseC(SessionHandler(sessionStore, "posty-session"))

	// Chain for authenticated routes
	authedChain := xhandler.Chain{}
	authedChain = append(authedChain, baseChain...)
	authedChain.UseC(AuthenticatedFilter("/login"))

	// Chain for unauthenticated routes
	unauthedChain := xhandler.Chain{}
	unauthedChain = append(unauthedChain, baseChain...)
	unauthedChain.UseC(UnauthenticatedFilter("/wall"))

	// Router
	mux := web.New()
	mainContext := context.WithValue(context.Background(), "oidc", client)
	mux.Get("/wall", handle(mainContext, authedChain.HandlerC(xhandler.HandlerFuncC(Index))))
	mux.Get("/login", handle(mainContext, unauthedChain.HandlerC(xhandler.HandlerFuncC(Login))))
	mux.Get("/logout", handle(mainContext, authedChain.HandlerC(xhandler.HandlerFuncC(Logout))))
	mux.Get("/oauth2cb", handle(mainContext, unauthedChain.HandlerC(xhandler.HandlerFuncC(OAuth2Redirect))))
	log.Infof("Listening on %s", *listen)
	log.Fatal(http.ListenAndServe(":8080", gctx.ClearHandler(mux)))
}

func SessionHandler(store *sessions.CookieStore, name string) func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			session, err := store.Get(r, name)
			if err != nil {
				log.Infof("Could not decode session %q from %q: %s", name, r.RemoteAddr, err)
			}
			ctx = context.WithValue(ctx, "session", session)
			next.ServeHTTPC(ctx, w, r)
		})
	}
}

func AuthenticatedFilter(loginUrl string) func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			session, ok := ctx.Value("session").(*sessions.Session)
			if !ok {
				log.Error("Context without valid session")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			if _, ok := session.Values["user"]; !ok {
				log.Info("Handler: Is not loggedin")
				http.Redirect(w, r, loginUrl, http.StatusFound)
				return
			}
			next.ServeHTTPC(ctx, w, r)
		})
	}
}

func UnauthenticatedFilter(loggedInUrl string) func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			session, ok := ctx.Value("session").(*sessions.Session)
			if !ok {
				log.Error("Context without valid session")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			if _, ok := session.Values["user"]; ok {
				log.Info("Handler: Is loggedin")
				http.Redirect(w, r, loggedInUrl, http.StatusFound)
				return
			}
			next.ServeHTTPC(ctx, w, r)
		})
	}
}
func handle(ctx context.Context, handlerc xhandler.HandlerC) web.Handler {
	return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
		newctx := context.WithValue(ctx, "urlparams", c.URLParams)
		handlerc.ServeHTTPC(newctx, w, r)
	})
}

func Login(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Info("Handler: Login")
	c := ctx.Value("oidc").(*oidc.Client)
	oac, err := c.OAuthClient()
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
}

func Logout(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Info("Handler: Logout")
	session := ctx.Value("session").(*sessions.Session)
	delete(session.Values, "user")
	session.Save(r, w)

	http.Redirect(w, r, "/login", http.StatusFound)
}

func OAuth2Redirect(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	log.Info("Handler: OAuth2Redirect")
	c := ctx.Value("oidc").(*oidc.Client)
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "code query param must be set", http.StatusBadRequest)
		return
	}

	tok, err := c.ExchangeAuthCode(code)
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

	http.Redirect(w, r, "/wall", http.StatusFound)
}

func Index(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	params := ctx.Value("urlparams").(map[string]string)
	name := params["name"]
	select {
	case <-time.After(100 * time.Millisecond):
	case <-ctx.Done():
		http.Error(w, "Timeout", 400)
		return
	}
	fmt.Fprintf(w, "Welcome! %s\n", name)
}
