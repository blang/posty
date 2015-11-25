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
	"github.com/satori/go.uuid"
)

var plog *logrus.Entry

func init() {
	plog = logrus.New().WithFields(logrus.Fields{
		"env": "DynamoPostPeer",
	})
}

type DynamoPostPeer struct {
	model *DynamoModel
}

func (pp *DynamoPostPeer) GetByID(id string) (*model.Post, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String("post"),
		IndexName:              aws.String("IDIndex"),
		KeyConditionExpression: aws.String("id = :id"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":id": { // Required
				S: aws.String(id),
			},
		},
		Limit: aws.Int64(1),
	}
	respQuery, err := pp.model.db.Query(params)
	if err != nil {
		return nil, err
	}
	// Check if exactly one result
	if respQuery.Count == nil || *respQuery.Count != 1 || len(respQuery.Items) != 1 {
		return nil, fmt.Errorf("Results: %d len(%d)", respQuery.Count, respQuery.Items)
	}
	item := respQuery.Items[0]
	if item["wall_id"] == nil || item["created_at"] == nil {
		return nil, fmt.Errorf("Fields 'wall_id' or 'created_at' nil")
	}
	paramsQuery := &dynamodb.GetItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"wall_id": {
				S: item["wall_id"].S,
			},
			"created_at": {
				N: item["created_at"].N,
			},
		},
		TableName: aws.String("post"),
	}
	resp, err := pp.model.db.GetItem(paramsQuery)

	if err != nil {
		return nil, err
	}

	p := &model.Post{
		Peer: pp,
	}
	err = unmarshalPost(p, resp.Item)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (pp *DynamoPostPeer) NewPost(uid string) *model.Post {
	return &model.Post{
		Peer:      pp,
		ID:        uuid.NewV4().String(),
		UID:       uid,
		CreatedAt: time.Now(),
	}
}

func (pp *DynamoPostPeer) SaveNew(p *model.Post) error {
	if p == nil {
		return errors.New("Post is nil")
	}
	items := make(map[string]*dynamodb.AttributeValue)
	items["wall_id"] = &dynamodb.AttributeValue{
		S: aws.String("1"),
	}
	err := marshalPost(p, items)
	if err != nil {
		return err
	}
	params := &dynamodb.PutItemInput{
		Item:      items,
		TableName: aws.String("post"),
	}
	_, err = pp.model.db.PutItem(params)

	if err != nil {
		return err
	}

	return nil
}

func (pp *DynamoPostPeer) Remove(p *model.Post) error {
	params := &dynamodb.DeleteItemInput{
		Key: map[string]*dynamodb.AttributeValue{
			"wall_id": {
				S: aws.String("1"),
			},
			"created_at": {
				N: aws.String(strconv.FormatInt(p.CreatedAt.UnixNano(), 10)),
			},
		},
		TableName: aws.String("post"),
	}
	_, err := pp.model.db.DeleteItem(params)
	if err != nil {
		return err
	}
	return nil
}
func (pp *DynamoPostPeer) getPosts(lastKey map[string]*dynamodb.AttributeValue) ([]*model.Post, error) {
	params := &dynamodb.QueryInput{
		TableName:              aws.String("post"),
		KeyConditionExpression: aws.String("wall_id = :wid AND created_at <= :now"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":now": {
				N: aws.String(strconv.FormatInt(time.Now().Add(24*time.Hour).UnixNano(), 10)),
			},
			":wid": {
				S: aws.String("1"),
			},
		},
		ScanIndexForward: aws.Bool(false),
	}
	if lastKey != nil {
		params.ExclusiveStartKey = lastKey
	}
	resp, err := pp.model.db.Query(params)
	if err != nil {
		return nil, err
	}
	var err1 error
	posts := make([]*model.Post, 0, len(resp.Items))
	for _, postResp := range resp.Items {
		p := &model.Post{}
		err1 = unmarshalPost(p, postResp)
		if err1 != nil {
			plog.Warnf("Error unmarshal post: %#v", postResp)
			continue
		}
		posts = append(posts, p)
	}
	if resp.LastEvaluatedKey != nil {
		newposts, err := pp.getPosts(resp.LastEvaluatedKey)
		if err != nil {
			return nil, err
		}
		posts = append(posts, newposts...)
	}
	return posts, nil
}
func (pp *DynamoPostPeer) GetPosts() ([]*model.Post, error) {
	return pp.getPosts(nil)
}

func unmarshalPost(p *model.Post, items map[string]*dynamodb.AttributeValue) error {
	if p == nil {
		return errors.New("Undefined post")
	}
	if v, ok := items["id"]; ok {
		if v.S != nil {
			p.ID = *v.S
		}
	}
	if v, ok := items["uid"]; ok {
		if v.S != nil {
			p.UID = *v.S
		}
	}
	if v, ok := items["message"]; ok {
		if v.S != nil {
			p.Message = *v.S
		}
	}
	if v, ok := items["username"]; ok {
		if v.S != nil {
			p.Username = *v.S
		}
	}
	if v, ok := items["created_at"]; ok {
		if v.N != nil {
			ts64, err := strconv.ParseInt(*v.N, 10, 64)
			if err == nil {
				p.CreatedAt = time.Unix(0, ts64)
			} else {
				plog.Warnf("Unable to parse 'created_at' on %s: %s", items["id"], err)
			}
		}
	}
	return nil
}

func marshalPost(p *model.Post, items map[string]*dynamodb.AttributeValue) error {
	if p == nil {
		return errors.New("Undefined post")
	}
	items["id"] = &dynamodb.AttributeValue{S: aws.String(p.ID)}
	items["uid"] = &dynamodb.AttributeValue{S: aws.String(p.UID)}
	if p.Message != "" {
		items["message"] = &dynamodb.AttributeValue{S: aws.String(p.Message)}
	}
	if p.Username != "" {
		items["username"] = &dynamodb.AttributeValue{S: aws.String(p.Username)}
	}
	items["created_at"] = &dynamodb.AttributeValue{N: aws.String(strconv.FormatInt(p.CreatedAt.UnixNano(), 10))}

	return nil
}
