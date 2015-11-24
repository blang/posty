package controller

import (
	"fmt"
	"net/http"
	"posty/model"
	"posty/oidc"

	log "github.com/Sirupsen/logrus"
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
	Data         AuthDataProvider
	Provider     oidc.Provider
	ProviderName string
}

func NewAuthController(data AuthDataProvider, provider oidc.Provider, providerName string) *AuthController {
	return &AuthController{
		Data:         data,
		Provider:     provider,
		ProviderName: providerName,
	}
}

func (c *AuthController) Login() xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Login")
		c.Provider.NewAuth(w, r)
		return
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

func (c *AuthController) Callback(successURL string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Callback")
		uid, err := c.Provider.Callback(w, r)

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
		uuid := c.ProviderName + ":" + *uid

		u, err := c.loginUser(uuid)
		if err != nil {
			log.Warnf("Could not create new user: %s", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		session := ctx.Value("session").(*sessions.Session)
		session.Values["user"] = u.ID
		session.Save(r, w)

		http.Redirect(w, r, successURL, http.StatusFound)
	})
}

func (c *AuthController) loginUser(uuid string) (*model.User, error) {
	u, err := c.Data.GetByOAuthID(uuid)
	if u == nil || err != nil {
		u = c.Data.NewUser()
		u.OAuthID = uuid
		log.Infof("User to create: %#v", u)
		err = c.Data.SaveNew(u)
		if err != nil {
			return nil, fmt.Errorf("Could not save new user: %s", err)
		}
	}
	err = c.Data.UpdateLastLogin(u.ID)
	if err != nil {
		return nil, fmt.Errorf("Could not update last login: %s", err)
	}
	return u, nil
}
