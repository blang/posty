package model

import "time"

type PostPeer interface {
	GetByID(id string) (*Post, error)
	GetPosts() ([]*Post, error)
	NewPost(uid string) *Post
	SaveNew(p *Post) error
	Remove(p *Post) error
}

type Post struct {
	ID        string
	UID       string
	Message   string
	CreatedAt time.Time
	IsNew     bool
	Peer      PostPeer
}

func (p *Post) SaveNew() error {
	return p.Peer.SaveNew(p)
}

// ByCreatedAtDESC represents a sort interface for sorting Posts descendingby CreatedAt
type ByCreatedAtDESC []*Post

func (o ByCreatedAtDESC) Len() int           { return len(o) }
func (o ByCreatedAtDESC) Swap(i, j int)      { o[i], o[j] = o[j], o[i] }
func (o ByCreatedAtDESC) Less(i, j int) bool { return o[i].CreatedAt.After(o[j].CreatedAt) }
