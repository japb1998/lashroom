package database

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
	"github.com/japb1998/lashroom/scheduleEmail/pkg/client"
)

var (
	ClientTable = os.Getenv("CLIENT_TABLE")
)

type ClientRepository struct {
	Client *DynamoClient
}

type ClientEntity struct {
	PrimaryKey   string  `json:"primaryKey"`
	SortKey      string  `json:"sortKey"`
	Phone        *string `json:"phone"`
	Email        *string `json:"email"`
	ClientName   string  `json:"clientName"`
	CreatedAt    string  `json:"createdAt"`
	LastUpdateAt string  `json:"lastUpdateAt"`
	Description  string  `json:"description"`
}

func (c *ClientEntity) ToClientDto() client.ClientDto {
	id := c.SortKey

	return client.ClientDto{
		CreatedBy:   c.PrimaryKey,
		Id:          &id,
		Phone:       c.Phone,
		Email:       c.Email,
		ClientName:  c.ClientName,
		CreatedAt:   c.CreatedAt,
		Description: c.Description,
	}
}

func NewClientRepository() *ClientRepository {
	return &ClientRepository{
		Client: NewDynamoClient(),
	}
}

func (c *ClientRepository) GetClientsByCreator(createdBy string) ([]client.ClientDto, error) {

	queryValue, err := dynamodbattribute.MarshalMap(map[string]any{
		":primaryKey": createdBy,
	})

	if err != nil {
		log.Println(err)

		return nil, fmt.Errorf("error getting clients")
	}

	queryInput := &dynamodb.QueryInput{
		TableName:                 &ClientTable,
		KeyConditionExpression:    aws.String("#primaryKey = :primaryKey"),
		ExpressionAttributeValues: queryValue,
		ExpressionAttributeNames: map[string]*string{
			"#primaryKey": aws.String("primaryKey"),
		},
	}

	output, err := c.Client.Query(queryInput)

	if err != nil {
		log.Println(err)

		return nil, fmt.Errorf("error getting clients")
	}

	var clientEntityList []ClientEntity

	for _, item := range output.Items {
		var entity ClientEntity
		if err := dynamodbattribute.UnmarshalMap(item, &entity); err != nil {
			log.Println(err.Error())
		} else {
			clientEntityList = append(clientEntityList, entity)
		}

	}

	if err != nil {
		log.Println(err)
		return nil, fmt.Errorf("error getting clients")
	}

	var clientDtoList = make([]client.ClientDto, len(clientEntityList))

	for i, entity := range clientEntityList {

		clientDtoList[i] = entity.ToClientDto()

	}

	return clientDtoList, nil
}

func (c *ClientRepository) GetClientWithFilters(createdBy string, clientDto client.ClientDto) ([]client.ClientDto, error) {
	primaryKeyExpressionList := []string{"#primaryKey = :primaryKey"}

	attributeValues := map[string]any{
		":primaryKey": createdBy,
	}
	filterExpressionList := make([]string, 0)
	expressionAttributeNames := map[string]*string{
		"#primaryKey": aws.String("primaryKey"),
	}
	if clientDto.Id != nil {
		attributeValues[":id"] = *clientDto.Id
		expressionAttributeNames["#id"] = aws.String("id")
		primaryKeyExpressionList = append(primaryKeyExpressionList, "#id = :id")
	}
	if clientDto.Phone != nil {
		attributeValues[":phone"] = *clientDto.Phone
		expressionAttributeNames["#phone"] = aws.String("phone")
		filterExpressionList = append(filterExpressionList, "#phone = :phone")
	}
	if clientDto.Email != nil {
		attributeValues[":email"] = *clientDto.Email
		expressionAttributeNames["#email"] = aws.String("email")
		filterExpressionList = append(filterExpressionList, "#email = :email")
	}

	marshaledValues, err := dynamodbattribute.MarshalMap(attributeValues)

	if err != nil {

		log.Println(err.Error())

		return nil, errors.New("error while retreiving clients")
	}

	filterExpression := strings.Join(filterExpressionList, " and ")
	primaryKeyExpression := strings.Join(primaryKeyExpressionList, " and ")
	queryInput := &dynamodb.QueryInput{
		TableName:                 &ClientTable,
		KeyConditionExpression:    aws.String(primaryKeyExpression),
		ExpressionAttributeValues: marshaledValues,
		ExpressionAttributeNames:  expressionAttributeNames,
		FilterExpression:          &filterExpression,
	}

	output, err := c.Client.Query(queryInput)

	if err != nil {

		log.Println(err.Error())

		return nil, errors.New("error while retreiving clients")
	}

	var clientEntityList []ClientEntity

	err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &clientEntityList)

	if err != nil {

		log.Println(err.Error())

		return nil, errors.New("error while retreiving clients")
	}
	clientList := make([]client.ClientDto, len(clientEntityList))
	for i, clientEntity := range clientEntityList {
		clientList[i] = clientEntity.ToClientDto()
	}
	return clientList, nil
}

func (c *ClientRepository) CreateClient(clientDto client.ClientDto) (client.ClientDto, error) {
	id := uuid.New().String()
	clientDto.Id = &id
	ClientEntity := ClientEntity{
		PrimaryKey:   clientDto.CreatedBy,
		SortKey:      *clientDto.Id,
		ClientName:   clientDto.ClientName,
		Phone:        clientDto.Phone,
		Email:        clientDto.Email,
		CreatedAt:    time.Now().Format(time.RFC3339),
		LastUpdateAt: time.Now().Format(time.RFC3339),
		Description:  clientDto.Description,
	}

	item, err := dynamodbattribute.MarshalMap(ClientEntity)

	if err != nil {
		log.Println(err)
		return client.ClientDto{}, fmt.Errorf("error while creating client")
	}

	putItem := &dynamodb.PutItemInput{
		TableName: &ClientTable,
		Item:      item,
	}
	_, err = c.Client.PutItem(putItem)

	if err != nil {
		log.Println(err)
		return client.ClientDto{}, fmt.Errorf("error while creating client")
	}

	clientDto.CreatedAt = ClientEntity.CreatedAt
	clientDto.LastUpdateAt = ClientEntity.LastUpdateAt

	return clientDto, nil
}

func (c *ClientRepository) UpdateUser(createdBy string, clientId string, clientDto client.ClientDto) (client.ClientDto, error) {

	expression := "SET "
	expressionList := make([]string, 0)
	updateExpressionValues := make(map[string]string)
	updateExpressionNames := make(map[string]*string)

	if clientDto.ClientName != "" {
		updateExpressionValues[":clientName"] = clientDto.ClientName
		updateExpressionNames["#clientName"] = aws.String("clientName")
		expressionList = append(expressionList, "#clientName = :clientName")
	}
	if clientDto.Description != "" {
		updateExpressionValues[":description"] = clientDto.Description
		updateExpressionNames["#description"] = aws.String("description")
		expressionList = append(expressionList, "#description = :description")
	}
	if clientDto.Phone != nil {
		updateExpressionValues[":phone"] = *clientDto.Phone
		updateExpressionNames["#phone"] = aws.String("phone")
		expressionList = append(expressionList, "#phone = :phone")
	}
	if clientDto.Email != nil {
		updateExpressionValues[":email"] = *clientDto.Email
		updateExpressionNames["#email"] = aws.String("email")
		expressionList = append(expressionList, "#email = :email")
	}

	if len(expressionList) == 0 {
		return client.ClientDto{}, errors.New("empty update not allowed")
	}

	expression += fmt.Sprintf(" %v", strings.Join(expressionList, ", "))

	marshalledExpressionValues, err := dynamodbattribute.MarshalMap(updateExpressionValues)

	if err != nil {

		log.Println(err)
		return client.ClientDto{}, errors.New("error Marshalling update values")

	}

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    clientId,
	})

	if err != nil {

		log.Println(err)
		return client.ClientDto{}, errors.New("error Marshalling update values")

	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName:                 &ClientTable,
		Key:                       key,
		UpdateExpression:          &expression,
		ExpressionAttributeNames:  updateExpressionNames,
		ExpressionAttributeValues: marshalledExpressionValues,
		ReturnValues:              aws.String("ALL_NEW"),
	}

	output, err := c.Client.UpdateItem(updateInput)

	if err != nil {
		log.Println(output)
		return client.ClientDto{}, errors.New("error updating item")
	}

	var clientOuput ClientEntity
	fmt.Println(output.Attributes)
	if err := dynamodbattribute.UnmarshalMap(output.Attributes, &clientOuput); err != nil {
		return client.ClientDto{}, errors.New("error while marshalling updated value, value was possibly updated")
	}

	return clientOuput.ToClientDto(), nil
}

func (c *ClientRepository) DeleteClient(createdBy, id string) error {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    id,
	})

	input := &dynamodb.DeleteItemInput{
		TableName: &ClientTable,
		Key:       key,
	}
	if err != nil {
		log.Println(err.Error())
		return errors.New("error While deleting Client")
	}

	if _, err := c.Client.DeleteItem(input); err != nil {
		log.Println(err)
		return errors.New("error While Deleting Client")
	}

	return nil
}

func (c *ClientRepository) GetClientById(createdBy, id string) (*client.ClientDto, error) {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    id,
	})

	input := &dynamodb.GetItemInput{
		TableName: &ClientTable,
		Key:       key,
	}
	if err != nil {
		log.Println(err.Error())
		return nil, errors.New("error While Getting Client")
	}

	if item, err := c.Client.Client.GetItem(input); err != nil {
		log.Println(err)
		return nil, errors.New("error While Getting Client")
	} else {
		var clientEntity ClientEntity

		if len(item.Item) == 0 {
			return nil, nil
		}
		err := dynamodbattribute.UnmarshalMap(item.Item, &clientEntity)

		if err != nil {
			return nil, errors.New("error While Getting Client")
		}
		dto := clientEntity.ToClientDto()
		return &dto, nil
	}

}
