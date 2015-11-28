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

// AuthDataProvider defines a the needed model interactions
type AuthDataProvider interface {
	GetByOAuthID(oauthid string) (*model.User, error)
	UpdateLastLogin(id string) error
	NewUser() *model.User
	SaveNew(u *model.User) error
}

// AuthController handles login using oidc and logout.
type AuthController struct {
	Data         AuthDataProvider
	Provider     oidc.Provider
	ProviderName string
}

// NewAuthController creates a new instance associated with an oidc provider.
func NewAuthController(data AuthDataProvider, provider oidc.Provider, providerName string) *AuthController {
	return &AuthController{
		Data:         data,
		Provider:     provider,
		ProviderName: providerName,
	}
}

// Login handles login requests and delegates to the oidc provider.
func (c *AuthController) Login() xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Login")
		c.Provider.NewAuth(w, r)
		return
	})
}

// Logout handles logout requests and invalidates the users session.
func (c *AuthController) Logout(loginURL string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Logout")
		session := ctx.Value("session").(*sessions.Session)
		delete(session.Values, "user")
		session.Save(r, w)

		http.Redirect(w, r, loginURL, http.StatusFound)
	})
}

// Callback handles the oidc/oauth2 callback after a login attempt from the user.
// If the idenity provider returned a proof for valid login, the userid is stored in the session.
// This includes the model lookup and a possible creation for new users.
// The users last login timestamp is updated.
func (c *AuthController) Callback(successURL string) xhandler.HandlerC {
	return xhandler.HandlerFuncC(func(ctx context.Context, w http.ResponseWriter, r *http.Request) {
		log.Info("Handler: Callback")
		user, err := c.Provider.Callback(w, r)

		if err != nil {
			log.Printf("Error occurred: %s", err)
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		if user == nil {
			log.Printf("Error occurred uid is nil")
			http.Error(w, "Bad Request", http.StatusBadRequest)
			return
		}
		uuid := c.ProviderName + ":" + user["id"]

		u, err := c.loginUser(uuid, user["name"])
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

// loginUser queries the database for the given uuid and otherwise creates a new user with the given username.
// It updates the users last login timestamp and returns the user data.
func (c *AuthController) loginUser(uuid, username string) (*model.User, error) {
	u, err := c.Data.GetByOAuthID(uuid)
	if u == nil || err != nil {
		u = c.Data.NewUser()
		u.OAuthID = uuid
		u.Username = username
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
