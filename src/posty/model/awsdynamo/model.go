package awsdynamo

import (
	"posty/model"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

// DynamoModel implements `posty/model` for the dynamodb
type DynamoModel struct {
	db       *dynamodb.DynamoDB
	userPeer *DynamoUserPeer
	postPeer *DynamoPostPeer
}

// NewModelFromSession creates an new Model from an aws session.
func NewModelFromSession(s *session.Session) *DynamoModel {
	model := &DynamoModel{
		db: dynamodb.New(s),
	}
	model.userPeer = &DynamoUserPeer{
		model: model,
	}

	model.postPeer = &DynamoPostPeer{
		model: model,
	}
	return model
}

// UserPeer returns the dynamodb UserPeer associated with the model
func (m *DynamoModel) UserPeer() model.UserPeer {
	return m.userPeer
}

// PostPeer returns the dynamodb PostPeer associated with the model
func (m *DynamoModel) PostPeer() model.PostPeer {
	return model.PostPeer(m.postPeer)
}
