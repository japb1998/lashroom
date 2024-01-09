package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/japb1998/control-tower/internal/model"
)

var notificationRepo *notificationRepository
var notificationLogger = log.New(os.Stdout, "[Notification Repository] ", log.Default().Flags())
var (
	ErrEmptyUpdate          = errors.New("Update cannot have empty paremeters")
	ErrNotificationNotFound = errors.New("Notification not found")
)

type PaginationOps struct {
	Limit int
	Skip  int
}

type PaginatedNotifications struct {
	Total int64                    `json:"total"`
	Data  []model.NotificationItem `json:"data"`
}

func NewNotificationRepository(sess *session.Session) *notificationRepository {
	client := newDynamoClient(sess)
	if notificationRepo == nil {
		notificationRepo = &notificationRepository{
			client:    client,
			tableName: os.Getenv("EMAIL_TABLE"),
		}
	}
	return notificationRepo
}

type PatchNotificationItem struct {
	Date            time.Time `json:"date"`
	Status          string    `json:"status"`
	ClientId        string    `json:"clientId"`
	DeliveryMethods []int8    `json:"deliveryMethods"`
	TTL             int64     `json:"TTL"`
}

type notificationRepository struct {
	client    *DynamoClient
	tableName string
}

func (r *notificationRepository) SetStatus(partitionKey string, sortKey string, status string) error {
	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": partitionKey,
		"sortKey":    sortKey,
	})
	if err != nil {
		return err
	}
	statusAttr, err := dynamodbattribute.Marshal(status)

	if err != nil {
		return err
	}

	input := &dynamodb.UpdateItemInput{
		TableName:        &r.tableName,
		Key:              key,
		UpdateExpression: aws.String("SET #status = :status"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": statusAttr,
		},
	}
	_, err = r.client.UpdateItem(input)

	if err != nil {
		return err
	}

	return nil
}

func (r *notificationRepository) Update(notification model.NotificationItem) (model.NotificationItem, error) {
	return model.NotificationItem{}, nil
}

// Create
func (r *notificationRepository) Create(notification model.NotificationItem) error {

	marshalledItem, err := dynamodbattribute.MarshalMap(notification)

	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		TableName: &r.tableName,
		Item:      marshalledItem,
	}
	_, err = r.client.PutItem(input)

	if err != nil {
		return err
	}
	return nil
}
func (r *notificationRepository) Delete(createdBy, name string) error {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    name,
	})

	if err != nil {
		log.Println(err.Error())
		return fmt.Errorf("error while deleting notification: %s", name)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key:       key,
	}

	_, err = r.client.DeleteItem(input)

	if err != nil {
		log.Println(err.Error())
		return fmt.Errorf("error while deleting notification: %s", name)
	}
	return nil
}

func (r *notificationRepository) GetNotification(createdBy, name string) (*model.NotificationItem, error) {
	sortKey, err := dynamodbattribute.Marshal(name)

	if err != nil {
		return nil, fmt.Errorf("invalid name provided name: '%s', error: %w", name, err)
	}

	partitionKey, err := dynamodbattribute.Marshal(createdBy)

	if err != nil {
		return nil, fmt.Errorf("invalid creator provided creator: '%s', error: %w", createdBy, err)
	}

	primaryKey := map[string]*dynamodb.AttributeValue{
		"sortKey":    sortKey,
		"primaryKey": partitionKey,
	}
	input := &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       primaryKey,
	}

	output, err := r.client.GetOne(input)

	if err != nil {
		return nil, fmt.Errorf("error while getting notification: %w", err)
	}
	var item model.NotificationItem

	if output.Item == nil || len(output.Item) == 0 {
		return nil, ErrNotificationNotFound
	}

	err = dynamodbattribute.UnmarshalMap(output.Item, &item)

	if err != nil {
		return nil, err
	}

	return &item, nil
}

// GetNotificationCountByCreator
func (r *notificationRepository) GetNotificationCountByCreator(createdBy string) (int64, error) {
	creatorAttr, err := dynamodbattribute.Marshal(createdBy)

	if err != nil {
		notificationLogger.Println(err)
		return 0, fmt.Errorf("error marshalling creator: %w", err)
	}

	var count int64
	var LastEvaluatedKey map[string]*dynamodb.AttributeValue
	input := &dynamodb.QueryInput{
		TableName:              &r.tableName,
		KeyConditionExpression: aws.String("#createdBy = :createdBy"),
		ExpressionAttributeNames: map[string]*string{
			"#createdBy": aws.String("primaryKey"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":createdBy": creatorAttr,
		},
		ExclusiveStartKey: LastEvaluatedKey,
		Select:            aws.String("COUNT"),
	}
	for {
		output, err := r.client.Query(input)
		if err != nil {
			return 0, fmt.Errorf("error querying notification by creator: %w", err)
		}
		count += *output.Count

		if output.LastEvaluatedKey == nil {
			break
		}
		LastEvaluatedKey = output.LastEvaluatedKey
	}
	return count, nil
}

// GetNotificationsByCreator gets notification by its creator. pagination is Zero based.
func (r *notificationRepository) GetNotificationsByCreator(createdBy string, ops *PaginationOps) (*PaginatedNotifications, error) {
	creatorAttr, err := dynamodbattribute.Marshal(createdBy)

	if err != nil {
		return nil, fmt.Errorf("error marshalling creator: %w", err)
	}

	if ops == nil {
		return nil, fmt.Errorf("Pagination ops is not optional")
	}

	var items = make([]model.NotificationItem, 0)
	var LastEvaluatedKey map[string]*dynamodb.AttributeValue
	input := &dynamodb.QueryInput{
		TableName:              &r.tableName,
		KeyConditionExpression: aws.String("#createdBy = :createdBy"),
		ExpressionAttributeNames: map[string]*string{
			"#createdBy": aws.String("primaryKey"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":createdBy": creatorAttr,
		},
		ExclusiveStartKey: LastEvaluatedKey,
		ScanIndexForward:  aws.Bool(true),
		IndexName:         aws.String("DATE"),
		Limit:             aws.Int64(int64(ops.Limit + ops.Skip)),
	}
	for {
		if len(items) != 0 {
			input.Limit = aws.Int64(int64(ops.Limit + ops.Skip - len(items)))
		}

		output, err := r.client.Query(input)
		if err != nil {
			return nil, fmt.Errorf("error querying notification by creator: %w", err)
		}
		var list []model.NotificationItem
		err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &list)

		if err != nil {
			return nil, fmt.Errorf("error querying notification by creator: %w", err)
		}
		if len(list) > 0 {
			items = append(items, list...)
		}
		if output.LastEvaluatedKey == nil || len(items) >= (ops.Skip+ops.Limit) {
			break
		}
		LastEvaluatedKey = output.LastEvaluatedKey
	}
	// manual pagination.
	if len(items) > ops.Skip+ops.Limit {
		items = items[ops.Skip : ops.Skip+ops.Limit]
	} else if len(items) <= ops.Skip {
		items = make([]model.NotificationItem, 0)
	} else {
		items = items[ops.Skip:]
	}

	count, err := r.GetNotificationCountByCreator(createdBy)

	if err != nil {
		return nil, err
	}
	paginatedRes := PaginatedNotifications{
		Total: count,
		Data:  items,
	}

	return &paginatedRes, nil

}

func (r *notificationRepository) UpdateNotification(createdBy, name string, notification PatchNotificationItem) (*model.NotificationItem, error) {
	attrNames := map[string]*string{}
	attrValues := map[string]*dynamodb.AttributeValue{}
	updateExpSlc := make([]string, 0)
	if notification.ClientId != "" {
		updateExpSlc = append(updateExpSlc, fmt.Sprintf("#clientId = :clientId"))
		attrNames["#clientId"] = aws.String("clientId")

		val, err := dynamodbattribute.Marshal(notification.ClientId)
		if err != nil {
			return nil, fmt.Errorf("unable to update notification item error: %w", err)
		}

		attrValues[":clientId"] = val
	}

	if notification.Status != "" {
		updateExpSlc = append(updateExpSlc, fmt.Sprintf("#status = :status"))
		attrNames["#status"] = aws.String("status")

		val, err := dynamodbattribute.Marshal(notification.Status)
		if err != nil {
			return nil, fmt.Errorf("unable to update notification status error: %w", err)
		}

		attrValues[":status"] = val
	}

	if !notification.Date.IsZero() {
		updateExpSlc = append(updateExpSlc, fmt.Sprintf("#date = :date"))
		attrNames["#date"] = aws.String("date")

		val, err := dynamodbattribute.Marshal(notification.Date)
		if err != nil {
			return nil, fmt.Errorf("unable to update notification date error: %w", err)
		}

		attrValues[":date"] = val

		// we reset the ttl
		updateExpSlc = append(updateExpSlc, fmt.Sprintf("#ttl = :ttl"))
		attrNames["#ttl"] = aws.String("TTL")

		val, err = dynamodbattribute.Marshal(notification.Date.Add(time.Hour * 24))
		if err != nil {
			return nil, fmt.Errorf("unable to update notification date error: %w", err)
		}

		attrValues[":ttl"] = val
	}

	if notification.DeliveryMethods != nil || len(notification.DeliveryMethods) > 0 {
		updateExpSlc = append(updateExpSlc, fmt.Sprintf("#deliveryMethods = :deliveryMethods"))
		attrNames["#deliveryMethods"] = aws.String("deliveryMethods")

		val, err := dynamodbattribute.Marshal(notification.DeliveryMethods)
		if err != nil {
			return nil, fmt.Errorf("unable to update notification delivery methods error: %w", err)
		}

		attrValues[":deliveryMethods"] = val
	}

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    name,
	})
	if err != nil {
		return &model.NotificationItem{}, fmt.Errorf("Error unmarshalling Key error: %w", err)
	}
	exp := fmt.Sprintf("SET %s", strings.Join(updateExpSlc, ", "))

	if len(attrNames) == 0 || len(attrValues) == 0 {
		return &model.NotificationItem{}, ErrEmptyUpdate
	}
	input := &dynamodb.UpdateItemInput{
		TableName:                 &r.tableName,
		UpdateExpression:          aws.String(exp),
		ExpressionAttributeNames:  attrNames,
		ExpressionAttributeValues: attrValues,
		Key:                       key,
	}

	output, err := r.client.UpdateItem(input)

	if err != nil {
		notificationLogger.Printf("Error when updating item error: %s", err)

		return nil, fmt.Errorf("error when updating notification item error: %w", err)
	}
	var item model.NotificationItem
	err = dynamodbattribute.UnmarshalMap(output.Attributes, &item)

	if err != nil {
		return nil, fmt.Errorf("error when updating notification item error: %w", err)
	}
	return &item, nil
}
