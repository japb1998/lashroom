package database

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
)

var notificationTable = os.Getenv("EMAIL_TABLE")

type NotificationRepository struct {
	Client *DynamoClient
}

func (r *NotificationRepository) DeleteNotification(createdBy string, id string) error {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    id,
	})

	if err != nil {
		log.Println(err.Error())
		return fmt.Errorf("error while deleting notification: %s", id)
	}
	input := &dynamodb.DeleteItemInput{
		TableName: &notificationTable,
		Key:       key,
	}

	_, err = r.Client.DeleteItem(input)

	if err != nil {
		log.Println(err.Error())
		return fmt.Errorf("error while deleting notification: %s", id)
	}
	return nil
}
