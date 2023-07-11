package handler

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/japb1998/lashroom/clientQueue/pkg/operations"
	"github.com/japb1998/lashroom/clientQueue/pkg/record"
	"github.com/japb1998/lashroom/shared/pkg/database"
)

var tableName = os.Getenv("CLIENT_TABLE")
var queueUrl = os.Getenv("QUEUE_URL")

func Handler(ctx context.Context, sqsEvent events.SQSEvent) error {
	ddb := database.DynamoClient{
		Client: dynamodb.New(database.Session),
	}
	for _, messages := range sqsEvent.Records {

		defer func() {
			if r := recover(); r != nil {
				log.Println("Recovering ...")
			}
		}()

		err := processClientMessage(ddb, messages)

		if err != nil {
			return err
		}

	}
	return nil
}

func processClientMessage(ddb database.DynamoClient, messages events.SQSMessage) error {
	var ev record.Event
	processed := false

	defer func(processed *bool, receiptHandler *string) {

		if *processed {
			client, err := operations.NewSQSClient(nil, &queueUrl)

			if err != nil {
				log.Println("Error while creating new sqs client", err.Error())
			}

			client.DeleteMessage(receiptHandler)
		}
	}(&processed, &messages.ReceiptHandle)

	err := json.Unmarshal([]byte(messages.Body), &ev)
	if err != nil {
		log.Println(err.Error())
		return errors.New("error while parsing Event to json attribute")
	}
	email, err := dynamodbattribute.Marshal(ev.Body["email"])
	createdBy, createdByError := dynamodbattribute.Marshal(ev.Body["createdBy"])

	if err != nil {
		log.Println(err.Error())
		return errors.New("error while parsing email attribute")
	} else if createdByError != nil {
		log.Println(createdByError.Error())
		return errors.New("error while parsing createdBy attribute")
	}
	input := &dynamodb.QueryInput{

		TableName:              &tableName,
		KeyConditionExpression: aws.String("#createdBy = :createdBy"),
		FilterExpression:       aws.String("#email = :email"),
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":email":     email,
			":createdBy": createdBy,
		},
		ExpressionAttributeNames: map[string]*string{
			"#email":     aws.String("email"),
			"#createdBy": aws.String("primaryKey"),
		},
	}
	output, err := ddb.Query(input)

	if err != nil {
		log.Println(err.Error())
		return errors.New("error Scanning for Clients")
	}

	if len(output.Items) > 0 {
		log.Printf("Client Already Exist under %v \n", ev.Body)
		return nil
	}
	var clientEmail *string
	var clientPhone *string

	if res, ok := ev.Body["email"]; ok {
		switch t := res.(type) {
		case string:
			clientEmail = &t
		}
	}
	if res, ok := ev.Body["phone"]; ok {
		switch t := res.(type) {
		case string:
			clientPhone = &t
		}
	}

	newClient := database.ClientEntity{
		PrimaryKey: ev.Body["createdBy"].(string),
		SortKey:    uuid.New().String(),
		Email:      clientEmail,
		Phone:      clientPhone,
		ClientName: ev.Body["clientName"].(string),
	}
	item, err := dynamodbattribute.MarshalMap(newClient)

	if err != nil {
		log.Println(err.Error())
		return errors.New("error Marshalling Item")
	}
	putItemInput := &dynamodb.PutItemInput{
		TableName: &tableName,
		Item:      item,
	}
	_, err = ddb.PutItem(putItemInput)

	if err != nil {
		log.Println(err.Error())
		return errors.New("error while saving item")
	}

	log.Printf("Successfully Created Client %v \n", ev.Body)
	return nil
}
