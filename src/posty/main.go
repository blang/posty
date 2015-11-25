package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	filepath "path"
	"posty/controller"
	"posty/middleware"
	"posty/model"
	"posty/model/awsdynamo"
	"posty/oidc"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	gctx "github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

const envprefix = "POSTY_"

func envOrDefault(env string, def string) string {
	if ev := os.Getenv(envprefix + env); ev != "" {
		return ev
	}
	return def
}

var (
	listen                 = flag.String("http", envOrDefault("LISTEN", ":8080"), "Listen on")
	awsprofile             = flag.String("awsprofile", envOrDefault("AWS_PROFILE", ""), "AWS Profile using shared credential file")
	frontendPath           = flag.String("frontend-path", envOrDefault("FRONTEND_PATH", "./frontend"), "Path to frontend")
	debug                  = flag.Bool("debug", false, "Enable debugging")
	oidcGoogleClientID     = flag.String("oidc-google-client-id", envOrDefault("OIDC_GOOGLE_CLIENT_ID", ""), "Google OpenID Connect Client ID")
	oidcGoogleClientSecret = flag.String("oidc-google-client-secret", envOrDefault("OIDC_GOOGLE_CLIENT_SECRET", ""), "Google OpenID Connect Client Secret")
	oidcPaypalClientID     = flag.String("oidc-paypal-client-id", envOrDefault("OIDC_PAYPAL_CLIENT_ID", ""), "Paypal OpenID Connect Client ID")
	oidcPaypalClientSecret = flag.String("oidc-paypal-client-secret", envOrDefault("OIDC_PAYPAL_CLIENT_SECRET", ""), "Paypal OpenID Connect Client Secret")
	publicURL              = flag.String("public-url", envOrDefault("PUBLIC_URL", "http://127.0.0.1:8080"), "http://[host]")
	sessionHashKey         = flag.String("session-hash-key", envOrDefault("SESSION_HASH_KEY", ""), "Session hash key, 32/64 Byte")
	sessionBlockKey        = flag.String("session-block-key", envOrDefault("SESSION_BLOCK_KEY", ""), "Session block encryption key, valid lengths are 16, 24, or 32 bytes to select AES-128, AES-192, or AES-256")
)

func checkFlags() bool {
	flag.Parse()
	if *listen == "" {
		log.Fatal("Flag 'listen' must be set")
		return false
	}
	if *frontendPath == "" {
		log.Fatal("Flag 'frontend-path' must be set")
		return false
	}
	if *oidcGoogleClientID == "" {
		log.Fatal("Flag 'oidc-google-client-id' must be set")
		return false
	}
	if *oidcGoogleClientSecret == "" {
		log.Fatal("Flag 'oidc-google-client-secret' must be set")
		return false
	}
	if *oidcPaypalClientID == "" {
		log.Fatal("Flag 'oidc-paypal-client-id' must be set")
		return false
	}
	if *oidcPaypalClientSecret == "" {
		log.Fatal("Flag 'oidc-paypal-client-secret' must be set")
		return false
	}
	if *publicURL == "" {
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

type postDataProvider struct {
	model.PostPeer
	UserPeer model.UserPeer
}

func (p *postDataProvider) GetUserByID(id string) (*model.User, error) {
	return p.UserPeer.GetByID(id)
}

func main() {
	if !checkFlags() {
		os.Exit(1)
	}

	sessionStore := sessions.NewCookieStore([]byte(*sessionHashKey), []byte(*sessionBlockKey))

	// OpenID Connect Providers

	// Google
	oidcGoogleLoginRoute := "/logingoogle"
	oidcGoogleCBRoute := "/gcallback"
	oidcGoogle := &oidc.Google{
		ClientID:     *oidcGoogleClientID,
		ClientSecret: *oidcGoogleClientSecret,
		RedirectURI:  *publicURL + oidcGoogleCBRoute,
		SessionStore: sessionStore,
	}

	// PayPal
	oidcPaypalLoginRoute := "/loginpaypal"
	oidcPaypalCBRoute := "/pcallback"
	oidcPaypal := &oidc.Paypal{
		ClientID:     *oidcPaypalClientID,
		ClientSecret: *oidcPaypalClientSecret,
		RedirectURI:  *publicURL + oidcPaypalCBRoute,
		SessionStore: sessionStore,
	}

	// Dynamodb
	cfg := &aws.Config{
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String("http://localhost:8000"),
		Credentials: credentials.NewSharedCredentials("", *awsprofile),
	}
	sess := session.New(cfg)
	if *debug {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}

	// Model
	var m model.Model
	m = awsdynamo.NewModelFromSession(sess)

	// Controller
	// OAuth / OpenID Connect
	authCGoogle := controller.NewAuthController(m.UserPeer(), oidcGoogle, "google")
	authCPaypal := controller.NewAuthController(m.UserPeer(), oidcPaypal, "paypal")

	// Post Controller
	postContrData := &postDataProvider{
		PostPeer: m.PostPeer(),
		UserPeer: m.UserPeer(),
	}
	postController := &controller.PostController{
		Model: postContrData,
	}

	// Middleware
	baseChain := xhandler.Chain{}
	baseChain.UseC(xhandler.TimeoutHandler(2 * time.Second))

	// Session management
	sessionMiddleware := middleware.Session{}
	sessionMiddleware.Init([]byte(*sessionHashKey), []byte(*sessionBlockKey))
	baseChain.UseC(sessionMiddleware.Enable("posty-session"))

	// Chain for authenticated routes
	authedChain := xhandler.Chain{}
	authedChain = append(authedChain, baseChain...)
	authedChain.UseC(middleware.AuthenticatedFilter("/login"))
	authedChain.UseC(middleware.UserContext())

	// Chain for authenticated routes with json response
	jsonChain := xhandler.Chain{}
	jsonChain = append(jsonChain, authedChain...)
	jsonChain.UseC(middleware.JSONWrapper())

	// Chain for unauthenticated routes
	unauthedChain := xhandler.Chain{}
	unauthedChain = append(unauthedChain, baseChain...)
	unauthedChain.UseC(middleware.UnauthenticatedFilter("/"))

	// Main Context
	ctx := context.Background()
	route := func(chain xhandler.Chain, handler xhandler.HandlerC) web.Handler {
		return handle(ctx, chain.HandlerC(handler))
	}

	// Routes
	mux := web.New()
	mux.Get("/api/posts", route(jsonChain, xhandler.HandlerFuncC(postController.Posts)))
	mux.Post("/api/posts", route(jsonChain, xhandler.HandlerFuncC(postController.Create)))
	mux.Delete("/api/posts/:id", route(jsonChain, xhandler.HandlerFuncC(postController.Remove)))
	// OIDC Routes
	mux.Get(oidcGoogleLoginRoute, route(unauthedChain, authCGoogle.Login()))
	mux.Get(oidcGoogleCBRoute, route(unauthedChain, authCGoogle.Callback("/")))
	mux.Get(oidcPaypalLoginRoute, route(unauthedChain, authCPaypal.Login()))
	mux.Get(oidcPaypalCBRoute, route(unauthedChain, authCPaypal.Callback("/")))
	mux.Get("/logout", route(authedChain, authCGoogle.Logout("/login")))

	// Static file
	mux.Get("/login", route(unauthedChain, serveSingleFile(filepath.Join(*frontendPath, "login.html"))))
	mux.Get("/", route(authedChain, serveSingleFile(filepath.Join(*frontendPath, "index.html"))))
	mux.Get("/static/*", route(baseChain, serveFiles(filepath.Join(*frontendPath, "/static"), "/static/")))

	log.Infof("Listening on %s", *listen)
	log.Fatal(http.ListenAndServe(":8080", gctx.ClearHandler(mux)))
}

// handler transformation xhandler.HandlerC -> web.Handler
func handle(ctx context.Context, handlerc xhandler.HandlerC) web.Handler {
	return web.HandlerFunc(func(c web.C, w http.ResponseWriter, r *http.Request) {
		newctx := context.WithValue(ctx, "urlparams", c.URLParams)
		handlerc.ServeHTTPC(newctx, w, r)
	})
}

func Index() xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		params := ctx.Value("urlparams").(map[string]string)
		name := params["name"]
		select {
		case <-time.After(100 * time.Millisecond):
		case <-ctx.Done():
			http.Error(w, "Timeout", 400)
			return
		}
		fmt.Fprintf(w, "Welcome! %s\n", name)
	})
}

func serveSingleFile(path string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Infof("Serve static file: %s", path)
		http.ServeFile(w, r, path)
	})
}
func serveFiles(path string, prefix string) xhandler.HandlerC {
	fileserver := http.FileServer(http.Dir(path))
	handler := http.StripPrefix(prefix, fileserver)
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Infof("Serving: %s", r.RequestURI)
		handler.ServeHTTP(w, r)
	})
}
