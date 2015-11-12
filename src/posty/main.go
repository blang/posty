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
	"github.com/rs/xhandler"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

var (
	listen           = flag.String("http", ":8080", "Listen on")
	clientID         = flag.String("client-id", "", "OAuth client ID")
	clientSecret     = flag.String("client-secret", "", "OAuth client secret")
	oauthRedirectURL = flag.String("oauth-redirect-url", "http://127.0.0.1:8080/oauth2cb", "http://[host]/oauth2cb")
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

		sleep := 3 * time.Second
		log.Printf("failed fetching provider config, trying again in %v: %v", sleep, err)
		time.Sleep(sleep)
	}

	log.Printf("fetched provider config from %s: %#v", oauthDiscovery, cfg)

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
	c := xhandler.Chain{}
	c.UseC(xhandler.TimeoutHandler(2 * time.Second))
	// Router
	mux := web.New()
	mainContext := context.WithValue(context.Background(), "oidc", client)
	mux.Get("/test/:name", handle(mainContext, c.HandlerC(xhandler.HandlerFuncC(Index))))
	mux.Get("/login", handle(mainContext, c.HandlerC(xhandler.HandlerFuncC(Login))))
	mux.Get("/oauth2cb", handle(mainContext, c.HandlerC(xhandler.HandlerFuncC(OAuth2Redirect))))
	log.Infof("Listening on %s", *listen)
	log.Fatal(http.ListenAndServe(":8080", mux))
}

func handle(ctx context.Context, handlerc xhandler.HandlerC) web.Handler {
	return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
		newctx := context.WithValue(ctx, "urlparams", c.URLParams)
		handlerc.ServeHTTPC(newctx, w, r)
	})
}

func Login(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	c := ctx.Value("oidc").(*oidc.Client)
	oac, err := c.OAuthClient()
	if err != nil {
		panic("unable to proceed")
	}

	u, err := url.Parse(oac.AuthCodeURL("", "", ""))
	if err != nil {
		panic("unable to proceed")
	}
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func OAuth2Redirect(ctx context.Context, w http.ResponseWriter, r *http.Request) {
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

	s := fmt.Sprintf("claims: %v", claims)
	w.Write([]byte(s))
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
