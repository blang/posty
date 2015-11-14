package integrationtest

import (
	"flag"
	"fmt"
	"os"
	"posty/model"
	"posty/model/awsdynamo"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var awsprofile = flag.String("profile", os.Getenv("AWS_PROFILE"), "AWS Profile using shared credential file")
var integration = flag.Bool("integration", false, "Enable integration tests")
var dynamodebug = flag.Bool("dynamodebug", false, "Enable for debug out of dynamo requests")
var cfg *aws.Config
var sess *session.Session

func TestMain(m *testing.M) {
	flag.Parse()
	if !*integration {
		fmt.Fprintln(os.Stderr, "Skipping integration tests")
		os.Exit(0)
	}
	cfg = &aws.Config{
		Region:      aws.String("us-west-2"),
		Endpoint:    aws.String("http://localhost:8000"),
		Credentials: credentials.NewSharedCredentials("", *awsprofile),
	}
	sess = session.New(cfg)
	if *dynamodebug {
		sess.Config.LogLevel = aws.LogLevel(aws.LogDebug)
	}

	if err := loadUserFixtures(sess); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading 'user' integration fixtures: %s", err)
		os.Exit(1)
	}
	if err := loadPostFixtures(sess); err != nil {
		fmt.Fprintf(os.Stderr, "Error loading 'post' integration fixtures: %s", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

var mmodel model.Model

func setup() {
	mmodel = awsdynamo.NewModelFromSession(sess)
}

func teardown() {

}

func deleteTable(db *dynamodb.DynamoDB, table string) error {
	params := &dynamodb.DeleteTableInput{
		TableName: aws.String(table),
	}
	_, err := db.DeleteTable(params)
	if err != nil {
		return err
	}
	return nil

}
