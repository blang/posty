package controller

import (
	"fmt"
	"posty/model"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockAuthDataProvider struct {
	getByOAuthIDFn    func(oauthid string) (*model.User, error)
	updateLastLoginFn func(id string) error
	newUserFn         func() *model.User
	saveNewFn         func(u *model.User) error
}

func (m *mockAuthDataProvider) GetByOAuthID(oauthid string) (*model.User, error) {
	return m.getByOAuthIDFn(oauthid)
}

func (m *mockAuthDataProvider) UpdateLastLogin(id string) error {
	return m.updateLastLoginFn(id)
}

func (m *mockAuthDataProvider) NewUser() *model.User {
	return m.newUserFn()
}

func (m *mockAuthDataProvider) SaveNew(u *model.User) error {
	return m.saveNewFn(u)
}

func TestAuthLoginGoogle(t *testing.T) {
	assert := assert.New(t)
	var updateCalled string
	ts := time.Unix(1448272067, 0)
	mock := &mockAuthDataProvider{
		getByOAuthIDFn: func(oauthid string) (*model.User, error) {

			return &model.User{
				OAuthID:   oauthid,
				ID:        "uid123",
				Username:  "oldusername",
				CreatedAt: ts,
			}, nil
		},
		updateLastLoginFn: func(id string) error {
			updateCalled = id
			return nil
		},
	}
	ac := &AuthController{
		Data: mock,
	}

	u, err := ac.loginUser("google:123", "newusername")
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	assert.Equal("uid123", u.ID)
	assert.Equal("oldusername", u.Username)
	assert.Equal("uid123", updateCalled)
	assert.Equal("google:123", u.OAuthID)
	assert.Equal(ts.Unix(), u.CreatedAt.Unix(), "CreatedAt does not match")
}

func TestAuthLoginGoogleCreateUser(t *testing.T) {
	assert := assert.New(t)
	var updateCalled string
	ts := time.Unix(1448272067, 0)
	var saveUser *model.User
	mock := &mockAuthDataProvider{
		getByOAuthIDFn: func(oauthid string) (*model.User, error) {
			return nil, fmt.Errorf("Unknown user")
		},
		updateLastLoginFn: func(id string) error {
			updateCalled = id
			return nil
		},
		newUserFn: func() *model.User {
			return &model.User{
				ID:        "uid123",
				CreatedAt: ts,
			}
		},
		saveNewFn: func(u *model.User) error {
			saveUser = u
			return nil
		},
	}
	ac := &AuthController{
		Data: mock,
	}

	u, err := ac.loginUser("google:123", "username")
	if err != nil {
		t.Fatalf("Error: %s", err)
	}
	assert.Equal("uid123", saveUser.ID)
	assert.Equal("uid123", u.ID)
	assert.Equal("username", u.Username)
	assert.Equal("uid123", updateCalled)
	assert.Equal("google:123", u.OAuthID)
	assert.Equal(ts.Unix(), u.CreatedAt.Unix(), "CreatedAt does not match")
	assert.Equal("uid123", updateCalled)
}
