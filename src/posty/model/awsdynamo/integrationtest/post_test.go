package integrationtest

import (
	"fmt"
	"posty/model"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func loadPostFixtures(s *session.Session) error {
	db := dynamodb.New(sess)
	if err := deleteTable(db, "post"); err != nil {
		fmt.Printf("Warn: Delete table 'post' failed: %s\n", err)
	}
	if err := createPostTable(db); err != nil {
		fmt.Printf("Warn: Create Post table failed: %s\n", err)
	}
	if err := fixturePost(db); err != nil {
		return err
	}
	return nil
}

func createPostTable(db *dynamodb.DynamoDB) error {
	params := &dynamodb.CreateTableInput{
		TableName: aws.String("post"),
		KeySchema: []*dynamodb.KeySchemaElement{
			{ // Required
				AttributeName: aws.String("wall_id"),
				KeyType:       aws.String("HASH"),
			},
			{ // Required
				AttributeName: aws.String("created_at"),
				KeyType:       aws.String("RANGE"),
			},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("uid"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("created_at"),
				AttributeType: aws.String("N"),
			},
			{
				AttributeName: aws.String("wall_id"),
				AttributeType: aws.String("S"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{
				IndexName: aws.String("UIDIndex"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("uid"),
						KeyType:       aws.String("HASH"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("INCLUDE"),
					NonKeyAttributes: []*string{
						aws.String("id"),
					},
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(1),
					WriteCapacityUnits: aws.Int64(1),
				},
			},
			{ // Required
				IndexName: aws.String("IDIndex"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{
						AttributeName: aws.String("id"),
						KeyType:       aws.String("HASH"),
					},
				},
				Projection: &dynamodb.Projection{
					ProjectionType: aws.String("INCLUDE"),
					NonKeyAttributes: []*string{
						aws.String("wall_id"),
						aws.String("created_at"),
						aws.String("uid"),
					},
				},
				ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
					ReadCapacityUnits:  aws.Int64(1),
					WriteCapacityUnits: aws.Int64(1),
				},
			},
		},
	}
	_, err := db.CreateTable(params)
	if err != nil {
		return err
	}
	return nil
}

func fixturePost(db *dynamodb.DynamoDB) error {
	params := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String("pid123"),
			},
			"uid": {
				S: aws.String("uid123"),
			},
			"wall_id": {
				S: aws.String("1"),
			},
			"message": {
				S: aws.String("message"),
			},
			"created_at": {
				N: aws.String(strconv.FormatInt(time.Now().UnixNano(), 10)),
			},
		},
		TableName: aws.String("post"),
	}
	_, err := db.PutItem(params)
	if err != nil {
		return err
	}

	return nil
}

func TestPostGetByID(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.PostPeer()
	p, err := peer.GetByID("pid123")
	if err != nil {
		t.Fatalf("Error getting ByID: %s\n", err)
	}

	if p == nil {
		t.Fatalf("Post is nil\n")
	}
	assert.Equal("pid123", p.ID)
	assert.Equal("uid123", p.UID)
	assert.Equal("message", p.Message)
	assert.True(p.CreatedAt.Unix() > 0, "Timestamp should exist")
}

func TestPostCreateNew(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.PostPeer()
	p := peer.NewPost("uid123")
	p.Message = "mymessage"
	err := p.SaveNew()
	if err != nil {
		t.Fatalf("Error saving new post: %s\n", err)
	}

	gp, err := peer.GetByID(p.ID)
	if err != nil {
		t.Fatalf("Could not get new created user: %s\n", err)
	}
	assert.Equal(p.ID, gp.ID)
	assert.Equal(p.UID, gp.UID)
	assert.Equal(p.Message, gp.Message)
	assert.Equal(p.CreatedAt.Unix(), gp.CreatedAt.Unix())
}

func TestPostRemove(t *testing.T) {
	setup()
	peer := mmodel.PostPeer()
	var err error
	p := peer.NewPost("uiddelete")
	err = p.SaveNew()
	if err != nil {
		t.Fatalf("Could not create post: %s", err)
	}

	err = peer.Remove(p)
	if err != nil {
		t.Fatalf("Could not remove post: %s", err)
	}

	// Test if entry exists
	gp, err := peer.GetByID(p.ID)
	if err == nil && gp != nil {
		t.Fatalf("Post still exists after remove")
	}
}

func TestPostGetPosts(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.PostPeer()
	var err error
	for i := 0; i < 1000; i++ {
		p := peer.NewPost("uidnew")
		p.Message = strings.Repeat("test", 1000)
		err = p.SaveNew()
		if err != nil {
			t.Logf("Error inserting post: %s", err)
		}
	}
	posts, err := peer.GetPosts()
	if err != nil {
		t.Fatalf("Error: %s\n", err)
	}

	// Filter only added posts
	posts = filterPosts(posts, func(p *model.Post) bool {
		return p.UID == "uidnew"
	})
	assert.True(len(posts) == 1000, fmt.Sprintf("Length of posts %d should be 1000", len(posts)))
	err = checkPosts(posts, func(p1, p2 *model.Post) error {
		if !p1.CreatedAt.After(p2.CreatedAt) {
			return fmt.Errorf("CreatedAt not ordered %s %s", p1.CreatedAt, p2.CreatedAt)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Order error: %s", err)
	}

	// Check for dups
	sort.Sort(model.ByCreatedAtDESC(posts))
	err = checkPosts(posts, func(p1, p2 *model.Post) error {
		if p1.CreatedAt == p2.CreatedAt {
			return fmt.Errorf("CreatedAt the same on %d", p1.CreatedAt)
		}
		return nil
	})
	if err != nil {
		t.Errorf("Found duplication: %s", err)
	}
}

func filterPosts(ps []*model.Post, fn func(*model.Post) bool) []*model.Post {
	var newps []*model.Post
	for _, p := range ps {
		if fn(p) {
			newps = append(newps, p)
		}
	}
	return newps
}

// Slice must be sorted
func checkPosts(ps []*model.Post, fn func(*model.Post, *model.Post) error) error {
	var last *model.Post
	var err error
	for _, p := range ps {
		if last != nil {
			if err = fn(last, p); err != nil {
				return err
			}
		}
		last = p
	}
	return nil
}
