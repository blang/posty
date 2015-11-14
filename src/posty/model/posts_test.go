package model

import (
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestSortByCreatedAtDesc(t *testing.T) {
	assert := assert.New(t)
	var posts []*Post
	posts = append(posts, &Post{CreatedAt: time.Now()})
	posts = append(posts, &Post{CreatedAt: time.Now().Add(time.Hour)})
	posts = append(posts, &Post{CreatedAt: time.Now().Add(time.Second)})
	sort.Sort(ByCreatedAtDESC(posts))
	var last *Post
	for _, p := range posts {
		if last != nil {
			assert.True(last.CreatedAt.After(p.CreatedAt))
		}
		last = p
	}
}
