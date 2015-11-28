package middleware

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

// AuthenticatedFilter filters logged in users. Users with an invalid session are redirected to the loginURL.
func AuthenticatedFilter(loginURL string) func(next xhandler.HandlerC) xhandler.HandlerC {
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
				http.Redirect(w, r, loginURL, http.StatusFound)
				return
			}
			next.ServeHTTPC(ctx, w, r)
		})
	}
}

// UnauthenticatedFilter filters not-logged in users. Users with a valid session are redirected to the loggedInURL.
func UnauthenticatedFilter(loggedInURL string) func(next xhandler.HandlerC) xhandler.HandlerC {
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
				http.Redirect(w, r, loggedInURL, http.StatusFound)
				return
			}
			next.ServeHTTPC(ctx, w, r)
		})
	}
}

// UserContext adds the users session data to the handler context.
func UserContext() func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			session, ok := ctx.Value("session").(*sessions.Session)
			if !ok {
				log.Error("Context without valid session")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			user, ok := session.Values["user"]
			if !ok {
				log.Error("Context without valid session")
				http.Error(w, "Something went wrong", http.StatusInternalServerError)
				return
			}
			ctx = context.WithValue(ctx, "user", user)
			next.ServeHTTPC(ctx, w, r)
		})
	}
}
