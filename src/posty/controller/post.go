package controller

import (
	"encoding/json"
	"net/http"
	"posty/model"

	log "github.com/Sirupsen/logrus"
	"golang.org/x/net/context"
)

// PostDataProvider defines the needed model interactions.
type PostDataProvider interface {
	GetUserByID(id string) (*model.User, error)
	GetPosts() ([]*model.Post, error)
	NewPost(uid string) *model.Post
	SaveNew(p *model.Post) error
	GetByID(id string) (*model.Post, error)
	Remove(p *model.Post) error
}

// PostController handles post related requests.
type PostController struct {
	Model PostDataProvider
}

type postsResponse struct {
	Data []*jsonPost `json:"data"`
}

type jsonPost struct {
	ID        string `json:"id"`
	UID       string `json:"user_id"`
	Username  string `json:"username"`
	Message   string `json:"message"`
	CreatedAt int64  `json:"created_at"`
}

// Posts gets all posts from the database and returns valid json, otherwise a json error.
func (p *PostController) Posts(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	ps, err := p.Model.GetPosts()
	if err != nil {
		jsonError(w, r, cErrServer, "")
		return
	}
	jsonPosts := make([]*jsonPost, len(ps))
	for i, p := range ps {
		jsonPosts[i] = &jsonPost{
			ID:        p.ID,
			UID:       p.UID,
			Username:  p.Username,
			Message:   p.Message,
			CreatedAt: p.CreatedAt.Unix(),
		}
	}
	resp := postsResponse{
		Data: jsonPosts,
	}
	enc := json.NewEncoder(w)
	err = enc.Encode(&resp)
	if err != nil {
		jsonError(w, r, cErrServer, "")
		return
	}
}

type postCreateReq struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

type postCreateResp struct {
	Data *jsonPost `json:"data"`
}

// Create handles a request to create a new post.
//
// Example request: `{"data":{"message":"test message"}}`
//
// It checks for a non empty message with a length of at least 6 characters.
// On success it inserts an new post into the model and returns the created object as json with status code `http.StatusCreated`.
// Otherwise a json error is returned.
func (p *PostController) Create(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user, ok := ctx.Value("user").(string)
	if !ok {
		log.Warnf("Invalid user context")
		jsonError(w, r, cErrServer, "")
		return
	}
	dec := json.NewDecoder(r.Body)
	defer r.Body.Close()
	var req postCreateReq
	err := dec.Decode(&req)
	if err != nil {
		jsonError(w, r, cErrClient, "")
		return
	}
	if req.Data.Message == "" || len(req.Data.Message) < 6 {
		jsonError(w, r, cErrClient, "Message too short")
		return
	}
	userdata, err := p.Model.GetUserByID(user)
	if err != nil {
		jsonError(w, r, cErrServer, "")
		return
	}
	post := p.Model.NewPost(user)
	post.Message = req.Data.Message
	post.Username = userdata.Username
	err = p.Model.SaveNew(post)
	if err != nil {
		log.Warnf("Could not save post: %s", err)
		jsonError(w, r, cErrServer, "")
	}
	jsonPost := &jsonPost{
		ID:        post.ID,
		UID:       post.UID,
		Username:  post.Username,
		Message:   post.Message,
		CreatedAt: post.CreatedAt.Unix(),
	}
	postCreateResp := postCreateResp{
		Data: jsonPost,
	}
	enc := json.NewEncoder(w)
	w.WriteHeader(http.StatusCreated)
	err = enc.Encode(postCreateResp)
	if err != nil {
		log.Warnf("Could not save post: %s", err)
		jsonError(w, r, cErrServer, "")
	}
}

// Remove handles post remove requests and removes the post from the model if the user id matches the logged in user.
// The post id is defined as an url parameter.
//
// On success an empty response with status http.StatusNoContent is written.
// If the post identified by the id could not be found http.StatusNotFound is returned.
// If the user id does not match http.StatusUnauthorized is returned.
func (p *PostController) Remove(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	user, ok := ctx.Value("user").(string)
	if !ok {
		log.Warnf("Invalid user context")
		jsonError(w, r, cErrServer, "")
		return
	}
	urlParams := ctx.Value("urlparams").(map[string]string)
	id, ok := urlParams["id"]
	if !ok {
		jsonError(w, r, cErrClient, "Missing id parameter")
		return
	}
	post, err := p.Model.GetByID(id)
	if err != nil {
		jsonError(w, r, http.StatusNotFound, "Resource not found")
		return
	}
	if post.UID != user {
		jsonError(w, r, http.StatusUnauthorized, "Not allowed to delete resource")
		return
	}
	err = p.Model.Remove(post)
	if err != nil {
		jsonError(w, r, cErrServer, "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
