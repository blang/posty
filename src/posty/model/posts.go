package model

import "time"

// PostPeer defines interactions with the post data.
type PostPeer interface {
	GetByID(id string) (*Post, error)
	GetPosts() ([]*Post, error)
	NewPost(uid string) *Post
	SaveNew(p *Post) error
	Remove(p *Post) error
}

// Post represents a users post send to the board
type Post struct {
	ID        string
	UID       string
	Username  string
	Message   string
	CreatedAt time.Time
	IsNew     bool
	Peer      PostPeer
}

// SaveNew saves a new post to the model.
func (p *Post) SaveNew() error {
	return p.Peer.SaveNew(p)
}

// ByCreatedAtDESC represents a sort interface for sorting Posts descendingby CreatedAt
type ByCreatedAtDESC []*Post

// Len returns the amount of posts
func (o ByCreatedAtDESC) Len() int { return len(o) }

// Swap swaps two items in the slice
func (o ByCreatedAtDESC) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

// Less defines the comparator of posts
func (o ByCreatedAtDESC) Less(i, j int) bool { return o[i].CreatedAt.After(o[j].CreatedAt) }
