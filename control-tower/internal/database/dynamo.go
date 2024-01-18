package database

import (
	"context"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

type DynamoClient struct {
	Client *dynamodb.DynamoDB
}

func newDynamoClient(sess *session.Session) *DynamoClient {

	client := dynamodb.New(sess, sess.Config.WithDisableSSL(false))
	return &DynamoClient{
		Client: client,
	}
}

func (dynamodb *DynamoClient) Query(queryInput *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	output, err := dynamodb.Client.QueryWithContext(ctx, queryInput)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return output, nil
}

func (dynamodb *DynamoClient) GetOne(input *dynamodb.GetItemInput) (*dynamodb.GetItemOutput, error) {

	output, err := dynamodb.Client.GetItem(input)

	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return output, nil
}

func (dynamodb *DynamoClient) PutItem(input *dynamodb.PutItemInput) (*dynamodb.PutItemOutput, error) {

	if output, err := dynamodb.Client.PutItem(input); err != nil {
		log.Println(err.Error())
		return nil, err
	} else {
		return output, nil
	}

}

func (dynamodb *DynamoClient) Scan(input *dynamodb.ScanInput) (*dynamodb.ScanOutput, error) {
	if output, err := dynamodb.Client.Scan(input); err != nil {

		log.Println(err.Error())
		return nil, err
	} else {
		return output, nil
	}
}

func (dynamodb *DynamoClient) UpdateItem(input *dynamodb.UpdateItemInput) (*dynamodb.UpdateItemOutput, error) {
	if output, err := dynamodb.Client.UpdateItem(input); err != nil {

		log.Println(err.Error())
		return nil, err
	} else {
		return output, nil
	}
}

func (dynamodb *DynamoClient) DeleteItem(input *dynamodb.DeleteItemInput) (*dynamodb.DeleteItemOutput, error) {

	if output, err := dynamodb.Client.DeleteItem(input); err != nil {
		log.Println(err.Error())
		return nil, err
	} else {
		return output, nil
	}
}
