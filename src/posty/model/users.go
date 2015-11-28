package model

import "time"

// UserPeer defines interactions with the user data.
type UserPeer interface {
	GetByID(id string) (*User, error)
	GetByOAuthID(id string) (*User, error)
	UpdateLastLogin(id string) error
	NewUser() *User
	SaveNew(user *User) error
}

// User represents an user in the model.
type User struct {
	ID        string
	OAuthID   string
	Email     string
	Username  string
	Peer      UserPeer
	CreatedAt time.Time
	LastLogin time.Time
}

// SaveNew saves a new user to the model.
func (u *User) SaveNew() error {
	return u.Peer.SaveNew(u)
}
