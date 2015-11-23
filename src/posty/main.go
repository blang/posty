package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"posty/controller"
	"posty/middleware"
	"posty/model"
	"posty/model/awsdynamo"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	gctx "github.com/gorilla/context"
	"github.com/gorilla/securecookie"
	"github.com/rs/xhandler"
	"github.com/zenazn/goji/web"
	"golang.org/x/net/context"
)

var (
	listen           = flag.String("http", ":8080", "Listen on")
	awsprofile       = flag.String("awsprofile", os.Getenv("AWS_PROFILE"), "AWS Profile using shared credential file")
	debug            = flag.Bool("debug", false, "Enable debugging")
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
	authController := controller.AuthController{
		Data: m.UserPeer(),
	}
	if err := authController.InitOAuth(*clientID, *clientSecret, oauthDiscovery, *oauthRedirectURL); err != nil {
		log.Fatalf("Error initializing OAuth: %s", err)
	}
	postController := &controller.PostController{
		Model: m.PostPeer(),
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
	unauthedChain.UseC(middleware.UnauthenticatedFilter("/wall"))

	// Main Context
	ctx := context.Background()
	route := func(chain xhandler.Chain, handler xhandler.HandlerC) web.Handler {
		return handle(ctx, chain.HandlerC(handler))
	}

	// Routes
	mux := web.New()
	mux.Get("/wall", route(authedChain, Index()))
	mux.Get("/api/posts", route(jsonChain, xhandler.HandlerFuncC(postController.Posts)))
	mux.Post("/api/posts", route(jsonChain, xhandler.HandlerFuncC(postController.Create)))
	mux.Delete("/api/posts/:id", route(jsonChain, xhandler.HandlerFuncC(postController.Remove)))
	mux.Get("/login", route(unauthedChain, authController.Login()))
	mux.Get("/logout", route(authedChain, authController.Logout("/login")))
	mux.Get("/oauth2cb", route(unauthedChain, authController.OAuth2Callback("/wall")))

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
