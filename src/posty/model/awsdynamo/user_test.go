package awsdynamo

import (
	"posty/model"
	"strconv"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/stretchr/testify/assert"
)

func TestUnmarshalUser(t *testing.T) {
	assert := assert.New(t)
	ts := time.Now()
	items := make(map[string]*dynamodb.AttributeValue)
	items["id"] = &dynamodb.AttributeValue{S: aws.String("uid123")}
	items["oauthid"] = &dynamodb.AttributeValue{S: aws.String("google:1234")}
	items["email"] = &dynamodb.AttributeValue{S: aws.String("test@example.com")}
	items["username"] = &dynamodb.AttributeValue{S: aws.String("username")}
	items["lastlogin"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(ts.Unix(), 10))}
	items["created_at"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(ts.Unix(), 10))}
	var u model.User
	err := unmarshalUser(&u, items)
	if err != nil {
		t.Fatalf("Error unmarshalling user: %s", err)
	}
	assert.Equal("uid123", u.ID)
	assert.Equal("google:1234", u.OAuthID)
	assert.Equal("test@example.com", u.Email)
	assert.Equal("username", u.Username)
	assert.Equal(ts.Unix(), u.LastLogin.Unix())
	assert.Equal(ts.Unix(), u.CreatedAt.Unix())
}

func TestMarshalUser(t *testing.T) {
	assert := assert.New(t)
	u := &model.User{}
	u.ID = "uid123"
	u.OAuthID = "google:1234"
	u.Email = "test@example.com"
	u.Username = "username"
	u.LastLogin = time.Now()
	u.CreatedAt = time.Now().Add(-time.Hour)
	m := make(map[string]*dynamodb.AttributeValue)
	err := marshalUser(u, m)
	if err != nil {
		t.Fatalf("Error marshalling user: %s", err)
	}
	awsValueString := func(k string) string {
		if v, ok := m[k]; ok {
			if v != nil && v.S != nil {
				return *v.S
			}
			t.Errorf("Key is nil: %s\n", k)
			return ""
		}
		t.Errorf("Invalid key: %s\n", k)
		return ""
	}
	awsValueInt64 := func(k string) int64 {
		if v, ok := m[k]; ok {
			if v != nil && v.N != nil {
				n, err := strconv.ParseInt(*v.N, 10, 64)
				if err != nil {
					t.Errorf("Could not parse int: %s\n", k)
					return 0
				}
				return n
			}
			t.Errorf("Key is nil: %s\n", k)
			return 0
		}
		t.Errorf("Invalid key: %s\n", k)
		return 0
	}
	assert.Equal(u.ID, awsValueString("id"))
	assert.Equal(u.OAuthID, awsValueString("oauthid"))
	assert.Equal(u.Email, awsValueString("email"))
	assert.Equal(u.Username, awsValueString("username"))
	assert.Equal(u.LastLogin.Unix(), awsValueInt64("lastlogin"))
	assert.Equal(u.CreatedAt.Unix(), awsValueInt64("created_at"))
}

func TestNewUser(t *testing.T) {
	assert := assert.New(t)
	p := &DynamoUserPeer{}
	u := p.NewUser()
	assert.NotNil(u)
	assert.True(len(u.ID) > 20)
	assert.True(u.CreatedAt.Unix() > time.Now().Add(-time.Hour).Unix())
	assert.True(u.CreatedAt.Unix() <= time.Now().Unix())
}
