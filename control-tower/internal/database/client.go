package database

import (
	"errors"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"

	"github.com/japb1998/control-tower/internal/model"
)

var (
	clientHandler = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("repository", "client")})
	clientLogger  = slog.New(clientHandler)
)

var clientRepository *ClientRepository

type ClientRepository struct {
	Client    *DynamoClient
	tableName string
}

type PatchClientItem struct {
	Phone       string     `json:"phone"`
	Email       string     `json:"email"`
	FirstName   string     `json:"firstName"`
	LastName    string     `json:"lastName"`
	LastSeen    *time.Time `json:"lastSeen"`
	Description string     `json:"description"`
	OptIn       *bool      `json:"optIn"`
}

func NewClientRepo(sess *session.Session) *ClientRepository {

	if clientRepository != nil {
		return clientRepository
	}
	client := newDynamoClient(sess)
	clientRepository = &ClientRepository{
		Client:    client,
		tableName: os.Getenv("CLIENT_TABLE"),
	}

	return clientRepository
}

func (c *ClientRepository) GetClientsByCreator(createdBy string) ([]model.ClientItem, error) {

	queryValue, err := dynamodbattribute.MarshalMap(map[string]any{
		":primaryKey": createdBy,
	})

	if err != nil {
		clientLogger.Error("error marshalling query key.", slog.String("error", err.Error()))

		return nil, fmt.Errorf("invalid creator: %s", createdBy)
	}

	clientEntityList := make([]model.ClientItem, 0)
	var lastEvaluatedKey map[string]*dynamodb.AttributeValue

	queryInput := &dynamodb.QueryInput{
		TableName:                 &c.tableName,
		KeyConditionExpression:    aws.String("#primaryKey = :primaryKey"),
		ExpressionAttributeValues: queryValue,
		ExpressionAttributeNames: map[string]*string{
			"#primaryKey": aws.String("primaryKey"),
		},
		ExclusiveStartKey: lastEvaluatedKey,
	}
	var items []model.ClientItem
	for {

		output, err := c.Client.Query(queryInput)
		if err != nil {
			temp := &dynamodb.ResourceNotFoundException{}
			if errors.As(err, &temp) {
				clientLogger.Debug("no items with the provided query")
				break
			}
			return nil, fmt.Errorf("error querying client items error: %s", err)
		}
		clientLogger.Debug("retrieved users.", slog.Int("length", len(output.Items)))
		err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &items)

		if err != nil {
			clientLogger.Error("error unmarshalling clients.", slog.String("error", err.Error()))
			return nil, fmt.Errorf("error unmarshalling clients")
		}

		clientEntityList = append(clientEntityList, items...)
		items = nil
		if output.LastEvaluatedKey == nil {
			break
		}
		queryInput.ExclusiveStartKey = output.LastEvaluatedKey

	}
	return clientEntityList, nil

}

func (c *ClientRepository) ClientCountWithFilters(createdBy string, clientPatch PatchClientItem) (int64, error) {
	var queryInput = &dynamodb.QueryInput{
		TableName: &c.tableName,
	}
	primaryKeyExpressionList := []string{"#primaryKey = :primaryKey"}

	attributeValues := map[string]any{
		":primaryKey": createdBy,
	}
	filterExpressionList := make([]string, 0)
	expressionAttributeNames := map[string]*string{
		"#primaryKey": aws.String("primaryKey"),
	}

	if clientPatch.Phone != "" {
		attributeValues[":phone"] = clientPatch.Phone
		expressionAttributeNames["#phone"] = aws.String("phone")
		filterExpressionList = append(filterExpressionList, "#phone = :phone")
	}
	if clientPatch.Email != "" {
		attributeValues[":email"] = clientPatch.Email
		expressionAttributeNames["#email"] = aws.String("email")
		filterExpressionList = append(filterExpressionList, "contains(#email, :email)")
	}

	if clientPatch.FirstName != "" {
		attributeValues[":firstName"] = clientPatch.FirstName
		expressionAttributeNames["#firstName"] = aws.String("firstName")
		filterExpressionList = append(filterExpressionList, "contains(#firstName,:firstName)")
	}

	if clientPatch.LastName != "" {
		attributeValues[":lastName"] = clientPatch.LastName
		expressionAttributeNames["#lastName"] = aws.String("lastName")
		filterExpressionList = append(filterExpressionList, "contains(#lastName, :lastName)")
	}

	if clientPatch.LastSeen != nil {
		attributeValues[":lastSeen"] = clientPatch.LastSeen
		expressionAttributeNames["#lastSeen"] = aws.String("lastSeen")
		filterExpressionList = append(filterExpressionList, "#lastSeen = :lastSeen")
	}

	marshaledValues, err := dynamodbattribute.MarshalMap(attributeValues)

	if err != nil {

		clientLogger.Error("failed to marshal values", slog.String("error", err.Error()))

		return 0, errors.New("error while retreiving clients")
	}
	if len(filterExpressionList) > 0 {
		queryInput.FilterExpression = aws.String(strings.Join(filterExpressionList, " OR "))
	}
	queryInput.KeyConditionExpression = aws.String(strings.Join(primaryKeyExpressionList, " and "))
	queryInput.ExpressionAttributeNames = expressionAttributeNames
	queryInput.ExpressionAttributeValues = marshaledValues
	queryInput.Select = aws.String("COUNT")
	output, err := c.Client.Query(queryInput)

	if err != nil {

		log.Println(err.Error())

		return 0, errors.New("error while retreiving clients")
	}

	return *output.Count, nil

}

func (c *ClientRepository) GetClientWithFilters(createdBy string, clientPatch PatchClientItem, p *PaginationOps) ([]model.ClientItem, error) {
	var clientEntityList []model.ClientItem

	var lastEvaluatedKey map[string]*dynamodb.AttributeValue
	var queryInput = &dynamodb.QueryInput{
		TableName:         &c.tableName,
		ExclusiveStartKey: lastEvaluatedKey,
		ScanIndexForward:  aws.Bool(true),
	}
	primaryKeyExpressionList := []string{"#primaryKey = :primaryKey"}

	attributeValues := map[string]any{
		":primaryKey": createdBy,
	}
	filterExpressionList := make([]string, 0)
	expressionAttributeNames := map[string]*string{
		"#primaryKey": aws.String("primaryKey"),
	}

	if clientPatch.Phone != "" {
		attributeValues[":phone"] = clientPatch.Phone
		expressionAttributeNames["#phone"] = aws.String("phone")
		filterExpressionList = append(filterExpressionList, "#phone = :phone")
	}
	if clientPatch.Email != "" {
		attributeValues[":email"] = clientPatch.Email
		expressionAttributeNames["#email"] = aws.String("email")
		filterExpressionList = append(filterExpressionList, "contains(#email, :email)")
	}

	if clientPatch.FirstName != "" {
		attributeValues[":firstName"] = clientPatch.FirstName
		expressionAttributeNames["#firstName"] = aws.String("firstName")
		filterExpressionList = append(filterExpressionList, "contains(#firstName,:firstName)")
	}

	if clientPatch.LastName != "" {
		attributeValues[":lastName"] = clientPatch.LastName
		expressionAttributeNames["#lastName"] = aws.String("lastName")
		filterExpressionList = append(filterExpressionList, "contains(#lastName, :lastName)")
	}

	if clientPatch.LastSeen != nil {

		attributeValues[":lastSeen"] = clientPatch.LastSeen
		expressionAttributeNames["#lastSeen"] = aws.String("lastSeen")
		filterExpressionList = append(filterExpressionList, "#lastSeen = :lastSeen")
	}
	// values
	marshaledValues, err := dynamodbattribute.MarshalMap(attributeValues)

	if err != nil {

		clientLogger.Error("error while marshalling values", slog.String("error", err.Error()))

		return nil, errors.New("error while retrieving clients")
	}
	queryInput.ExpressionAttributeValues = marshaledValues

	// names
	queryInput.ExpressionAttributeNames = expressionAttributeNames

	if len(filterExpressionList) > 0 {
		queryInput.FilterExpression = aws.String(strings.Join(filterExpressionList, " OR "))
	}

	queryInput.KeyConditionExpression = aws.String(strings.Join(primaryKeyExpressionList, " and "))
	var (
		output *dynamodb.QueryOutput
	)
	// loop until lastEvaluated key is nil or we reach our limit setup by the pagination.
	for {

		var clients []model.ClientItem
		// lastEvaluated key everytime we start. 1st is nil
		queryInput.ExclusiveStartKey = lastEvaluatedKey

		/* to be removed once performance issues are resolved */
		ts := time.Now()
		output, err = c.Client.Query(queryInput)
		fmt.Println(time.Since(ts))

		if err != nil {

			log.Println(err.Error())

			return nil, errors.New("error while retrieving clients")
		}

		err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &clients)

		if err != nil {

			clientLogger.Error("error while unmarshalling clients", slog.String("error", err.Error()))

			return nil, fmt.Errorf("error while retrieving clients error: %w", err)
		}

		clientEntityList = append(clientEntityList, clients...)

		if output.LastEvaluatedKey == nil || len(clientEntityList) >= p.Skip+p.Limit {
			break
		}

		lastEvaluatedKey = output.LastEvaluatedKey
	}

	// manual pagination.
	if len(clientEntityList) > p.Skip+p.Limit {
		clientEntityList = clientEntityList[p.Skip : p.Skip+p.Limit]
	} else if len(clientEntityList) <= p.Skip {
		clientEntityList = make([]model.ClientItem, 0)
	} else {
		clientEntityList = clientEntityList[p.Skip:]
	}

	return clientEntityList, nil

}

func (c *ClientRepository) CreateClient(client model.ClientItem) (model.ClientItem, error) {

	clientLogger.Debug("Creating user.", slog.Any("client", client))

	item, err := dynamodbattribute.MarshalMap(client)

	if err != nil {
		log.Println(err)
		return model.ClientItem{}, fmt.Errorf("error while creating client")
	}

	putItem := &dynamodb.PutItemInput{
		TableName: &c.tableName,
		Item:      item,
	}
	_, err = c.Client.PutItem(putItem)

	if err != nil {
		log.Println(err)
		return model.ClientItem{}, fmt.Errorf("error while creating client")
	}

	return model.ClientItem{}, nil
}

func (c *ClientRepository) UpdateUser(createdBy string, clientId string, client PatchClientItem) (model.ClientItem, error) {

	expression := "SET "
	expressionList := make([]string, 0)
	updateExpressionValues := make(map[string]any)
	updateExpressionNames := make(map[string]*string)

	if client.FirstName != "" {
		updateExpressionValues[":FirstName"] = client.FirstName
		updateExpressionNames["#FirstName"] = aws.String("firstName")
		expressionList = append(expressionList, "#FirstName = :FirstName")
	}
	if client.LastName != "" {
		updateExpressionValues[":LastName"] = client.LastName
		updateExpressionNames["#LastName"] = aws.String("lastName")
		expressionList = append(expressionList, "#LastName = :LastName")
	}
	if client.Description != "" {
		updateExpressionValues[":description"] = client.Description
		updateExpressionNames["#description"] = aws.String("description")
		expressionList = append(expressionList, "#description = :description")
	}
	if client.Phone != "" {
		updateExpressionValues[":phone"] = client.Phone
		updateExpressionNames["#phone"] = aws.String("phone")
		expressionList = append(expressionList, "#phone = :phone")
	}
	if client.Email != "" {
		updateExpressionValues[":email"] = client.Email
		updateExpressionNames["#email"] = aws.String("email")
		expressionList = append(expressionList, "#email = :email")
	}
	if client.OptIn != nil {
		updateExpressionValues[":optIn"] = *client.OptIn
		updateExpressionNames["#optIn"] = aws.String("optIn")
		expressionList = append(expressionList, "#optIn = :optIn")
	}

	if client.LastSeen != nil {
		updateExpressionValues[":lastSeen"] = client.LastSeen
		updateExpressionNames["#lastSeen"] = aws.String("lastSeen")
		expressionList = append(expressionList, "#lastSeen = :lastSeen")
	}

	if len(expressionList) == 0 {
		return model.ClientItem{}, fmt.Errorf("empty update not allowed")
	}
	expressionList = append(expressionList, fmt.Sprint("lastUpdateAt = :lastUpdateAt"))
	updateExpressionValues[":lastUpdateAt"] = time.Now().Format(time.RFC3339)
	expression += fmt.Sprintf(" %v", strings.Join(expressionList, ", "))

	marshalledExpressionValues, err := dynamodbattribute.MarshalMap(updateExpressionValues)

	if err != nil {

		log.Println(err)
		return model.ClientItem{}, fmt.Errorf("error Marshalling update values")

	}

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    clientId,
	})

	if err != nil {

		log.Println(err)
		return model.ClientItem{}, fmt.Errorf("error Marshalling update values")

	}

	updateInput := &dynamodb.UpdateItemInput{
		TableName:                 &c.tableName,
		Key:                       key,
		UpdateExpression:          &expression,
		ExpressionAttributeNames:  updateExpressionNames,
		ExpressionAttributeValues: marshalledExpressionValues,
		ReturnValues:              aws.String("ALL_NEW"),
	}

	output, err := c.Client.UpdateItem(updateInput)

	if err != nil {
		log.Println(output)
		return model.ClientItem{}, fmt.Errorf("error updating item error: %w", err)
	}

	var item model.ClientItem

	if err := dynamodbattribute.UnmarshalMap(output.Attributes, &item); err != nil {
		return model.ClientItem{}, fmt.Errorf("error while marshalling updated value, value was possibly updated error: %w", err)
	}
	return item, nil
}

func (c *ClientRepository) DeleteClient(createdBy, id string) error {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    id,
	})

	input := &dynamodb.DeleteItemInput{
		TableName: &c.tableName,
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

func (c *ClientRepository) GetClientById(createdBy, id string) (*model.ClientItem, error) {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"primaryKey": createdBy,
		"sortKey":    id,
	})

	input := &dynamodb.GetItemInput{
		TableName: &c.tableName,
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
		var clientEntity model.ClientItem

		if len(item.Item) == 0 {
			return nil, nil
		}
		err := dynamodbattribute.UnmarshalMap(item.Item, &clientEntity)

		if err != nil {
			return nil, errors.New("error While Getting Client")
		}

		return &clientEntity, nil
	}

}
