package database

import (
	"context"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/japb1998/control-tower/internal/model"
)

var connectionRepository *ConnectionRepository

type ConnectionRepository struct {
	Client    *DynamoClient
	tableName string
}

func NewConnectionRepo(sess *session.Session) *ConnectionRepository {

	clientLogger.Println("client Table ", os.Getenv("CONNECTION_TABLE"))
	if connectionRepository == nil {
		client := newDynamoClient(sess)
		connectionRepository = &ConnectionRepository{
			Client:    client,
			tableName: os.Getenv("CONNECTION_TABLE"),
		}
	}

	return connectionRepository
}

// SaveConnection stores the connection id and the user ID it belongs to.
func (cr *ConnectionRepository) SaveConnection(ctx context.Context, conn model.Connection) error {

	item, err := dynamodbattribute.MarshalMap(conn)

	if err != nil {
		return fmt.Errorf("unable to marshal connection error='%w'", err)
	}
	input := &dynamodb.PutItemInput{
		TableName: &cr.tableName,
		Item:      item,
	}

	_, err = cr.Client.PutItem(input)

	return err
}

// DeleteConnection - deletes connection from database
func (cr *ConnectionRepository) DeleteConnection(ctx context.Context, conn model.Connection) error {

	key, err := dynamodbattribute.MarshalMap(conn)

	if err != nil {
		return fmt.Errorf("unable to marshal connection error='%w'", err)
	}
	input := &dynamodb.DeleteItemInput{
		TableName: &cr.tableName,
		Key:       key,
	}

	_, err = cr.Client.DeleteItem(input)

	return err
}

// GetConnectionIds search for all connection Ids related to the specified userId.
func (cr *ConnectionRepository) GetConnectionIds(ctx context.Context, email string) ([]model.Connection, error) {

	keyExpression := aws.String("#email = :email")
	attrNames := map[string]*string{
		"#email": aws.String("email"),
	}
	dynamoKey, err := dynamodbattribute.Marshal(email)

	if err != nil {
		return nil, fmt.Errorf("invalid Email, error='%w'", err)
	}

	attrValues := map[string]*dynamodb.AttributeValue{
		":email": dynamoKey,
	}
	input := &dynamodb.QueryInput{
		KeyConditionExpression:    keyExpression,
		TableName:                 &cr.tableName,
		ExpressionAttributeNames:  attrNames,
		ExpressionAttributeValues: attrValues,
	}

	out, err := cr.Client.Query(input)

	if err != nil {
		return nil, fmt.Errorf("error while retrieving connection IDs error='%w'", err)
	}

	var connectionIds []model.Connection
	err = dynamodbattribute.UnmarshalListOfMaps(out.Items, &connectionIds)

	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall connectionIds from items, error:'%w'", err)
	}

	return connectionIds, nil
}
