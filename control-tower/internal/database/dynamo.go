package database

import (
	"log"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
)

var dynamoClient *DynamoClient

type DynamoClient struct {
	Client *dynamodb.DynamoDB
}

func newDynamoClient(sess *session.Session) *DynamoClient {
	if dynamoClient == nil {
		client := dynamodb.New(sess)
		return &DynamoClient{
			Client: client,
		}
	}
	return dynamoClient
}

func (dynamodb *DynamoClient) Query(queryInput *dynamodb.QueryInput) (*dynamodb.QueryOutput, error) {

	output, err := dynamodb.Client.Query(queryInput)

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
