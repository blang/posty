package integrationtest

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func loadUserFixtures(sess *session.Session) error {
	db := dynamodb.New(sess)
	if err := deleteTable(db, "user"); err != nil {
		fmt.Printf("Warn: Delete table 'user' failed: %s\n", err)
	}
	if err := createUserTable(db); err != nil {
		fmt.Printf("Warn: Create User table failed: %s\n", err)
	}
	if err := fixtureUser(db); err != nil {
		return err
	}
	return nil
}
func createUserTable(db *dynamodb.DynamoDB) error {
	params := &dynamodb.CreateTableInput{
		TableName: aws.String("user"),
		KeySchema: []*dynamodb.KeySchemaElement{
			{
				AttributeName: aws.String("id"),
				KeyType:       aws.String("HASH"),
			},
		},
		AttributeDefinitions: []*dynamodb.AttributeDefinition{
			{
				AttributeName: aws.String("id"),
				AttributeType: aws.String("S"),
			},
			{
				AttributeName: aws.String("oauthid"),
				AttributeType: aws.String("S"),
			},
		},
		ProvisionedThroughput: &dynamodb.ProvisionedThroughput{
			ReadCapacityUnits:  aws.Int64(1),
			WriteCapacityUnits: aws.Int64(1),
		},
		GlobalSecondaryIndexes: []*dynamodb.GlobalSecondaryIndex{
			{ // Required
				IndexName: aws.String("AuthIDIndex"),
				KeySchema: []*dynamodb.KeySchemaElement{
					{ // Required
						AttributeName: aws.String("oauthid"),
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
		},
	}
	_, err := db.CreateTable(params)
	if err != nil {
		return err
	}
	return nil
}

func fixtureUser(db *dynamodb.DynamoDB) error {
	params := &dynamodb.PutItemInput{
		Item: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String("uid123"),
			},
			"oauthid": {
				S: aws.String("google:1234"),
			},
			"email": {
				S: aws.String("test@example.com"),
			},
			"username": {
				S: aws.String("username"),
			},
			"created_at": {
				N: aws.String("123456789"),
			},
			"lastlogin": {
				N: aws.String("123456789"),
			},
		},
		TableName: aws.String("user"),
	}
	_, err := db.PutItem(params)

	if err != nil {
		return err
	}

	return nil
}

func TestUserGetByID(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.UserPeer()
	u, err := peer.GetByID("uid123")
	if err != nil {
		t.Fatalf("Error getting ByID: %s\n", err)
	}

	if u == nil {
		t.Fatalf("User is nil\n")
	}
	assert.Equal("uid123", u.ID)
	assert.Equal("google:1234", u.OAuthID)
	assert.Equal("test@example.com", u.Email)
	assert.Equal("username", u.Username)
}

func TestUserCreateNew(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.UserPeer()
	u := peer.NewUser()
	u.OAuthID = "google:5678"
	u.Username = "newuser"
	u.Email = "newuser@example.com"
	err := u.SaveNew()
	if err != nil {
		t.Fatalf("Error saving new user: %s\n", err)
	}

	gu, err := peer.GetByID(u.ID)
	if err != nil {
		t.Fatalf("Could not get new created user: %s\n", err)
	}
	assert.Equal(u.ID, gu.ID)
	assert.Equal(u.OAuthID, gu.OAuthID)
	assert.Equal(u.Username, gu.Username)
	assert.Equal(u.Email, gu.Email)
	assert.Equal(u.CreatedAt.Unix(), gu.CreatedAt.Unix())
}
func TestUserGetByOAuthID(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.UserPeer()
	u, err := peer.GetByOAuthID("google:1234")
	if err != nil {
		t.Fatalf("Error getting ByOAuthID: %s\n", err)
	}

	if u == nil {
		t.Fatalf("User is nil\n")
	}
	assert.Equal("uid123", u.ID)
}

func TestUserUpdateLastLogin(t *testing.T) {
	assert := assert.New(t)
	setup()
	peer := mmodel.UserPeer()
	err := peer.UpdateLastLogin("uid123")
	if err != nil {
		t.Fatalf("Could not update last login: %s\n", err)
	}

	u, err := peer.GetByID("uid123")
	if err != nil {
		t.Fatalf("Could not get new created user: %s\n", err)
	}
	assert.Equal("uid123", u.ID)
	assert.True(u.LastLogin.Unix() <= time.Now().Unix())
	assert.True(u.LastLogin.Unix() >= time.Now().Add(-time.Hour).Unix())
}
