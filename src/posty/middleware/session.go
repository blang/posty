package middleware

import (
	"net/http"

	log "github.com/Sirupsen/logrus"
	"github.com/gorilla/sessions"
	"github.com/rs/xhandler"
	"golang.org/x/net/context"
)

// Session middleware manages sessions
type Session struct {
	store sessions.Store
}

// Init creates a new session store
func (m *Session) Init(hashKey, blockKey []byte) {
	m.store = sessions.NewCookieStore(hashKey, blockKey)
}

// Enable enables session management. It creates a new session if none exists. Session is stored to the context as `session`.
func (m *Session) Enable(name string) func(next xhandler.HandlerC) xhandler.HandlerC {
	return func(next xhandler.HandlerC) xhandler.HandlerC {
		return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
			session, err := m.store.Get(r, name)
			if err != nil {
				log.Infof("Could not decode session %q from %q: %s", name, r.RemoteAddr, err)
			}
			ctx = context.WithValue(ctx, "session", session)
			next.ServeHTTPC(ctx, w, r)
		})
	}
}
