package awsdynamo

import (
	"posty/model"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoModel struct {
	db       *dynamodb.DynamoDB
	userPeer *DynamoUserPeer
	postPeer *DynamoPostPeer
}

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

func (m *DynamoModel) UserPeer() model.UserPeer {
	return m.userPeer
}

func (m *DynamoModel) PostPeer() model.PostPeer {
	return model.PostPeer(m.postPeer)
}
