package model

import "time"

type UserPeer interface {
	GetByID(id string) (*User, error)
	GetByOAuthID(id string) (*User, error)
	UpdateLastLogin(id string) error
	NewUser() *User
	SaveNew(user *User) error
}

type User struct {
	ID        string
	OAuthID   string
	Email     string
	Username  string
	Peer      UserPeer
	CreatedAt time.Time
	LastLogin time.Time
}

func (u *User) SaveNew() error {
	return u.Peer.SaveNew(u)
}
