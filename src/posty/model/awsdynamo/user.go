package awsdynamo

import (
	"errors"
	"fmt"
	"posty/model"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	uuid "github.com/satori/go.uuid"
)

var ulog *logrus.Entry

func init() {
	ulog = logrus.New().WithFields(logrus.Fields{
		"env": "DynamoUserPeer",
	})
}

type DynamoUserPeer struct {
	model *DynamoModel
}

func (p *DynamoUserPeer) GetByID(ID string) (*model.User, error) {
	params := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"id": { // Required
				S: aws.String(ID),
			},
		},
		TableName:      aws.String("user"),
		ConsistentRead: aws.Bool(true),
	}
	resp, err := p.model.db.GetItem(params)

	if err != nil {
		return nil, err
	}

	u := &model.User{
		Peer: p,
	}
	err = unmarshalUser(u, resp.Item)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (p *DynamoUserPeer) GetByOAuthID(ID string) (*model.User, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String("user"),
		IndexName:              aws.String("AuthIDIndex"),
		KeyConditionExpression: aws.String("oauthid = :aid"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":aid": {
				S: aws.String(ID),
			},
		},
	}

	resp, err := p.model.db.Query(params)
	if err != nil {
		return nil, err
	}

	// Check if exactly one result
	if resp.Count == nil || *resp.Count != 1 || len(resp.Items) != 1 {
		return nil, fmt.Errorf("Results: %d len(%d)", resp.Count, resp.Items)
	}

	u := &model.User{
		Peer: p,
	}
	err = unmarshalUser(u, resp.Items[0])
	if err != nil {
		return nil, err
	}
	return u, nil
}

func marshalUser(u *model.User, items map[string]*dynamodb.AttributeValue) error {
	if u == nil {
		return errors.New("Undefined user")
	}
	items["id"] = &dynamodb.AttributeValue{S: aws.String(u.ID)}
	items["oauthid"] = &dynamodb.AttributeValue{S: aws.String(u.OAuthID)}
	if u.Email != "" {
		items["email"] = &dynamodb.AttributeValue{S: aws.String(u.Email)}
	}
	if u.Username != "" {
		items["username"] = &dynamodb.AttributeValue{S: aws.String(u.Username)}
	}
	items["created_at"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(u.CreatedAt.Unix(), 10))}
	items["lastlogin"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(u.LastLogin.Unix(), 10))}

	return nil
}
func unmarshalUser(u *model.User, items map[string]*dynamodb.AttributeValue) error {
	if u == nil {
		return errors.New("Undefined user")
	}
	if v, ok := items["id"]; ok {
		if v.S != nil {
			u.ID = *v.S
		}
	}
	if v, ok := items["oauthid"]; ok {
		if v.S != nil {
			u.OAuthID = *v.S
		}
	}
	if v, ok := items["email"]; ok {
		if v.S != nil {
			u.Email = *v.S
		}
	}
	if v, ok := items["username"]; ok {
		if v.S != nil {
			u.Username = *v.S
		}
	}
	if v, ok := items["lastlogin"]; ok {
		if v.N != nil {
			ts64, err := strconv.ParseInt(*v.N, 10, 64)
			if err == nil {
				u.LastLogin = time.Unix(ts64, 0)
			} else {
				ulog.Warnf("Unable to parse 'lastlogin' on %s: %s", items["id"], err)
			}
		}
	}
	if v, ok := items["created_at"]; ok {
		if v.N != nil {
			ts64, err := strconv.ParseInt(*v.N, 10, 64)
			if err == nil {
				u.CreatedAt = time.Unix(ts64, 0)
			} else {
				ulog.Warnf("Unable to parse 'created_at' on %s: %s", items["id"], err)
			}
		}
	}
	return nil
}

func (p *DynamoUserPeer) NewUser() *model.User {
	return &model.User{
		Peer:      p,
		ID:        uuid.NewV4().String(),
		CreatedAt: time.Now(),
	}
}

// TODO: Check if uuid or oauthid already exists
func (p *DynamoUserPeer) SaveNew(u *model.User) error {
	if u == nil {
		return errors.New("User is nil")
	}
	items := make(map[string]*dynamodb.AttributeValue)
	err := marshalUser(u, items)
	if err != nil {
		return err
	}
	params := &dynamodb.PutItemInput{
		Item:      items,
		TableName: aws.String("user"),
	}
	_, err = p.model.db.PutItem(params)

	if err != nil {
		return err
	}

	return nil
}

func (p *DynamoUserPeer) UpdateLastLogin(id string) error {
	params := &dynamodb.UpdateItemInput{
		TableName: aws.String("user"),
		Key: map[string]*dynamodb.AttributeValue{
			"id": {
				S: aws.String(id),
			},
		},
		UpdateExpression: aws.String("SET lastlogin = :lastlogin"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":lastlogin": {
				N: aws.String(strconv.FormatInt(time.Now().Unix(), 10)),
			},
		},
		ReturnValues: aws.String("ALL_NEW"),
	}

	_, err := p.model.db.UpdateItem(params)
	if err != nil {
		return err
	}
	return nil
}
