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

func TestUnmarshalPost(t *testing.T) {
	assert := assert.New(t)
	ts := time.Now()
	items := make(map[string]*dynamodb.AttributeValue)
	items["id"] = &dynamodb.AttributeValue{S: aws.String("pid123")}
	items["uid"] = &dynamodb.AttributeValue{S: aws.String("uid123")}
	items["message"] = &dynamodb.AttributeValue{S: aws.String("message")}
	items["username"] = &dynamodb.AttributeValue{S: aws.String("username")}
	items["created_at"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(ts.UnixNano(), 10))}
	var p model.Post
	err := unmarshalPost(&p, items)
	if err != nil {
		t.Fatalf("Error unmarshalling user: %s", err)
	}
	assert.Equal("pid123", p.ID)
	assert.Equal("uid123", p.UID)
	assert.Equal("message", p.Message)
	assert.Equal("username", p.Username)
	assert.Equal(ts.UnixNano(), p.CreatedAt.UnixNano())
}

func TestMarshalPost(t *testing.T) {
	assert := assert.New(t)
	u := &model.Post{}
	u.ID = "pid123"
	u.UID = "uid123"
	u.Message = "message"
	u.Username = "username"
	u.CreatedAt = time.Now().Add(-time.Hour)
	m := make(map[string]*dynamodb.AttributeValue)
	err := marshalPost(u, m)
	if err != nil {
		t.Fatalf("Error marshalling post: %s", err)
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
	assert.Equal(u.UID, awsValueString("uid"))
	assert.Equal(u.Message, awsValueString("message"))
	assert.Equal(u.Username, awsValueString("username"))
	assert.Equal(u.CreatedAt.UnixNano(), awsValueInt64("created_at"))
}
