package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"posty/oidc"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
)

type OIDCStrategy interface {
	InitAuth(w http.ResponseWriter, r *http.Request)
	HandleCB(w http.ResponseWriter, r *http.Request) (uid *string, err error)
}

var provider = flag.String("provider", "google", "Provider: 'google' or 'paypal'")

func main() {
	flag.Parse()
	var client oidc.Provider
	if *provider == "google" {
		client = &oidc.Google{
			ClientID:     os.Getenv("OIDC_GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_GOOGLE_CLIENT_SECRET"),
			RedirectURI:  "http://127.0.0.1:8080/oauth2cb",
			SessionStore: sessions.NewCookieStore(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
		}
	} else if *provider == "paypal" {
		client = &oidc.Paypal{
			ClientID:     os.Getenv("OIDC_PAYPAL_CLIENT_ID"),
			ClientSecret: os.Getenv("OIDC_PAYPAL_CLIENT_SECRET"),
			RedirectURI:  "http://127.0.0.1:8080/oauth2cb",
			SessionStore: sessions.NewCookieStore(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32)),
		}
	} else {
		fmt.Fprintf(os.Stderr, "Invalid provider")
		os.Exit(1)
	}
	http.HandleFunc("/login", client.NewAuth)
	http.HandleFunc("/oauth2cb", func(w http.ResponseWriter, r *http.Request) {
		uid, err := client.Callback(w, r)
		if err != nil {
			log.Printf("Error occurred: %s", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if uid == nil {
			log.Printf("Error occurred uid is nil")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		fmt.Fprintf(w, "Uid is: %s", *uid)
	})

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Printf("Error: %s", err)
	}
}
