package controller

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"posty/model"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

type mockPostPeer struct {
	postsFn  func() ([]*model.Post, error)
	newFn    func(uid string) *model.Post
	saveFn   func(p *model.Post) error
	getidFn  func(id string) (*model.Post, error)
	removeFn func(p *model.Post) error
}

func (m *mockPostPeer) GetPosts() ([]*model.Post, error) {
	return m.postsFn()
}

func (m *mockPostPeer) NewPost(uid string) *model.Post {
	return m.newFn(uid)
}

func (m *mockPostPeer) SaveNew(p *model.Post) error {
	return m.saveFn(p)
}

func (m *mockPostPeer) Remove(p *model.Post) error {
	return m.removeFn(p)
}

func (m *mockPostPeer) GetByID(id string) (*model.Post, error) {
	return m.getidFn(id)
}

func TestPosts(t *testing.T) {
	assert := assert.New(t)
	const output = `{"data":[{"id":"id123","user_id":"uid123","message":"Message","created_at":1448272067}]}`
	ts := time.Unix(1448272067, 0)
	mockModel := &mockPostPeer{
		postsFn: func() ([]*model.Post, error) {
			return []*model.Post{
				{
					ID:        "id123",
					UID:       "uid123",
					Message:   "Message",
					CreatedAt: ts,
				},
			}, nil
		},
	}
	c := &PostController{
		Model: mockModel,
	}
	ctx := context.Background()
	w := httptest.NewRecorder()
	c.Posts(ctx, w, nil)
	assert.Equal(http.StatusOK, w.Code, "Invalid statuscode")
	assert.Equal(output, strings.TrimSpace(w.Body.String()), "Invalid output")
}
func TestCreate(t *testing.T) {
	assert := assert.New(t)
	const input = `{"data":{"message":"test message"}}`
	const output = `{"data":{"id":"id","user_id":"uid123","message":"test message","created_at":1448272067}}`
	ts := time.Unix(1448272067, 0)
	var post *model.Post
	mockModel := &mockPostPeer{
		newFn: func(uid string) *model.Post {
			return &model.Post{
				ID:        "id",
				UID:       uid,
				CreatedAt: ts,
			}
		},
		saveFn: func(p *model.Post) error {
			post = p
			return nil
		},
	}

	c := &PostController{
		Model: mockModel,
	}
	ctx := context.WithValue(context.Background(), "user", "uid123")
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "http://create", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Could not create request")
	}
	c.Create(ctx, w, r)
	assert.NotNil(post)
	assert.Equal("id", post.ID)
	assert.Equal("uid123", post.UID)
	assert.Equal(http.StatusCreated, w.Code, "Invalid statuscode")
	assert.Equal(output, strings.TrimSpace(w.Body.String()), "Invalid output")
}

func TestCreateInvalidJson(t *testing.T) {
	assert := assert.New(t)
	const input = `{"data":{"invalid":"test"}}`
	const output_partial = `{"errors":[{`
	mockModel := &mockPostPeer{}

	c := &PostController{
		Model: mockModel,
	}
	ctx := context.WithValue(context.Background(), "user", "uid123")
	w := httptest.NewRecorder()
	r, err := http.NewRequest("POST", "http://create", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Could not create request")
	}
	c.Create(ctx, w, r)
	assert.Equal(http.StatusBadRequest, w.Code, "Invalid statuscode")
	assert.True(strings.Contains(strings.TrimSpace(w.Body.String()), output_partial), fmt.Sprintf("Invalid output: %s", w.Body.String()))
}

func TestRemove(t *testing.T) {
	assert := assert.New(t)
	const input = `{"data":{"id":"postid"}}`
	var post *model.Post
	mockModel := &mockPostPeer{
		removeFn: func(p *model.Post) error {
			post = p
			return nil
		},
		getidFn: func(id string) (*model.Post, error) {
			assert.Equal("123", id, "ID must be '123'")
			return &model.Post{
				ID:  id,
				UID: "uid123",
			}, nil
		},
	}

	c := &PostController{
		Model: mockModel,
	}
	ctx := context.WithValue(context.Background(), "user", "uid123")
	ctx = context.WithValue(ctx, "urlparams", map[string]string{"id": "123"})
	w := httptest.NewRecorder()
	r, err := http.NewRequest("DELETE", "http://remove", strings.NewReader(input))
	if err != nil {
		t.Fatalf("Could not create request")
	}
	c.Remove(ctx, w, r)
	assert.NotNil(post)
	assert.Equal("123", post.ID)
	assert.Equal("uid123", post.UID)
	assert.Equal(http.StatusNoContent, w.Code, "Invalid statuscode")
	assert.Equal("", strings.TrimSpace(w.Body.String()), "Invalid output")

	// Unauthorized
	mockModel.getidFn = func(id string) (*model.Post, error) {
		assert.Equal("123", id, "ID must be '123'")
		return &model.Post{
			ID:  id,
			UID: "uid567",
		}, nil
	}
	w = httptest.NewRecorder()
	c.Remove(ctx, w, r)
	assert.Equal(http.StatusUnauthorized, w.Code, "Invalid statuscode")
	const unauth_error = `{"errors":[{"status":"401"`
	assert.True(strings.HasPrefix(strings.TrimSpace(w.Body.String()), unauth_error), "Invalid output")
}
