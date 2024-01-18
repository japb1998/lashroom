package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/japb1998/control-tower/internal/model"
)

var templateRepo *templateRepository

type templateRepository struct {
	client    *DynamoClient
	tableName string
	logger    *slog.Logger
}

func NewTemplateRepository(sess *session.Session) *templateRepository {
	if templateRepo == nil {
		tableName := os.Getenv("TEMPLATE_TABLE")
		loggerHandler := slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("repository", "template"), slog.String("tableName", tableName)})
		client := newDynamoClient(sess)
		templateRepo = &templateRepository{
			client:    client,
			tableName: tableName,
			logger:    slog.New(loggerHandler),
		}
	}
	fmt.Print("template repo init", templateRepo)
	return templateRepo
}

func (r *templateRepository) Create(ctx context.Context, t *model.TemplateItem) error {

	i, err := dynamodbattribute.MarshalMap(t)

	if err != nil {
		return err
	}
	input := &dynamodb.PutItemInput{
		TableName:    &r.tableName,
		Item:         i,
		ReturnValues: aws.String(dynamodb.ReturnValueAllOld),
	}

	_, err = r.client.PutItem(input)

	if err != nil {
		return err
	}

	r.logger.Info("template created")

	return nil
}

// Update - method to update template item.
func (r *templateRepository) Update(ctx context.Context, name, createdBy string, t *model.UpdateTemplate) (*model.TemplateItem, error) {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"name":      name,
		"createdBy": createdBy,
	})

	if err != nil {
		return nil, err
	}
	updateExpressionSlc := []string{}
	updateFieldValues := make(map[string]*dynamodb.AttributeValue)
	updateAttributeNames := make(map[string]*string)

	if t.Html != nil {
		updateExpressionSlc = append(updateExpressionSlc, "#html = :html")
		updateAttributeNames["#html"] = aws.String("html")

		val, err := dynamodbattribute.Marshal(t.Html)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal html value error=%w", err)
		}

		updateFieldValues[":html"] = val
	}

	if t.TemplateId != nil {
		updateExpressionSlc = append(updateExpressionSlc, "#templateId = :templateId")
		updateAttributeNames["#templateId"] = aws.String("templateId")

		val, err := dynamodbattribute.Marshal(t.TemplateId)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal templateId value error=%w", err)
		}

		updateFieldValues[":templateId"] = val
	}

	if t.Variables != nil {
		updateExpressionSlc = append(updateExpressionSlc, "#variables = :variables")
		updateAttributeNames["#variables"] = aws.String("variables")

		val, err := dynamodbattribute.Marshal(t.Variables)

		if err != nil {
			return nil, fmt.Errorf("failed to marshal variables value error=%w", err)
		}

		updateFieldValues[":variables"] = val
	}

	if len(updateExpressionSlc) == 0 {
		return nil, fmt.Errorf("empty update.")
	}
	updateExpression := fmt.Sprintf("SET %s", strings.Join(updateExpressionSlc, ", "))

	input := &dynamodb.UpdateItemInput{
		TableName:                 &r.tableName,
		Key:                       key,
		UpdateExpression:          &updateExpression,
		ExpressionAttributeNames:  updateAttributeNames,
		ExpressionAttributeValues: updateFieldValues,
	}

	out, err := r.client.UpdateItem(input)

	if err != nil {
		return nil, err
	}

	var i model.TemplateItem

	err = dynamodbattribute.UnmarshalMap(out.Attributes, &i)

	if err != nil {
		return nil, err
	}

	return &i, nil
}

// GetByKey gets template by composed key
func (r *templateRepository) GetByKey(ctx context.Context, creator string, name string) (*model.TemplateItem, error) {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"name":      name,
		"createdBy": creator,
	})

	if err != nil {
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: &r.tableName,
		Key:       key,
	}

	out, err := r.client.Client.GetItem(input)

	if out == nil || out.Item == nil {
		r.logger.Info("template not found", slog.Group("name", name, "creator", creator))
		return nil, nil
	}

	var t model.TemplateItem

	err = dynamodbattribute.UnmarshalMap(out.Item, &t)

	if err != nil {
		return nil, err
	}
	return &t, nil
}

// GetByCreator - gets all templates by creator.
func (r *templateRepository) GetByCreator(ctx context.Context, creator string, p *PaginationOps) ([]*model.TemplateItem, error) {

	creatorAttr, err := dynamodbattribute.Marshal(creator)

	if err != nil {
		return nil, fmt.Errorf("failed to marshal creator. error=%w", err)
	}
	templateItems := make([]*model.TemplateItem, 0)
	var lastEvaluatedKey map[string]*dynamodb.AttributeValue
	var queryInput = &dynamodb.QueryInput{
		TableName:              &r.tableName,
		ExclusiveStartKey:      lastEvaluatedKey,
		ScanIndexForward:       aws.Bool(true),
		KeyConditionExpression: aws.String("#createdBy = :createdBy"),
		ExpressionAttributeNames: map[string]*string{
			"#createdBy": aws.String("createdBy"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":createdBy": creatorAttr,
		},
	}

	// loop until lastEvaluated key is nil or we reach our limit setup by the pagination.
	for {
		var templates []*model.TemplateItem
		// lastEvaluated key everytime we start. 1st is nil
		queryInput.ExclusiveStartKey = lastEvaluatedKey

		output, err := r.client.Query(queryInput)

		if err != nil {

			r.logger.Error("failed to query templates", slog.String("error", err.Error()))

			return nil, errors.New("error while retrieving templates")
		}

		err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &templates)

		if err != nil {

			return nil, fmt.Errorf("error while retrieving templates error: %w", err)
		}

		templateItems = append(templateItems, templates...)

		if output.LastEvaluatedKey == nil || len(templateItems) >= p.Skip+p.Limit {
			break
		}

		lastEvaluatedKey = output.LastEvaluatedKey
	}

	return templateItems, nil
}

// GetTotalCount
func (r *templateRepository) GetTotalCount(ctx context.Context, creator string) (int64, error) {

	creatorAttr, err := dynamodbattribute.Marshal(creator)

	if err != nil {
		return 0, fmt.Errorf("failed to marshal creator. error=%w", err)
	}
	var count int64
	var lastEvaluatedKey map[string]*dynamodb.AttributeValue
	var queryInput = &dynamodb.QueryInput{
		TableName:              &r.tableName,
		ExclusiveStartKey:      lastEvaluatedKey,
		ScanIndexForward:       aws.Bool(true),
		KeyConditionExpression: aws.String("#createdBy = :createdBy"),
		ExpressionAttributeNames: map[string]*string{
			"#createdBy": aws.String("createdBy"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":createdBy": creatorAttr,
		},
		Select: aws.String("COUNT"),
	}

	// loop until lastEvaluated key is nil or we reach our limit setup by the pagination.
	for {
		output, err := r.client.Query(queryInput)
		if err != nil {
			return 0, fmt.Errorf("error querying notification by creator: %w", err)
		}
		count += *output.Count

		if output.LastEvaluatedKey == nil {
			break
		}
		lastEvaluatedKey = output.LastEvaluatedKey
	}
	return count, nil
}

// Delete receives name and creator and uses this as pk to delete a template.
func (r *templateRepository) Delete(ctx context.Context, name, creator string) error {

	key, err := dynamodbattribute.MarshalMap(map[string]string{
		"name":      name,
		"createdBy": creator,
	})

	if err != nil {
		return fmt.Errorf("invalid key error=%w", err)
	}

	input := &dynamodb.DeleteItemInput{
		TableName: &r.tableName,
		Key:       key,
	}

	_, err = r.client.DeleteItem(input)

	if err != nil {
		return fmt.Errorf("failed to delete template error=%w", err)
	}

	return nil
}
