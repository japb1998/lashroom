package main

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt"
	"github.com/japb1998/eliemail/dbmodule"
	"github.com/japb1998/eliemail/shared"
)

type Response events.APIGatewayProxyResponse

var ginLambda *ginadapter.GinLambda

var (
	TableName = os.Getenv("EMAIL_TABLE")
)

func serve() {
	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Gin cold start")
	r := gin.Default()
	schedule := r.Group("/schedule")
	schedule.GET("", func(c *gin.Context) {

		userEmail, err := getCreatedBy(c.Request.Header["Authorization"][0])
		if err != nil {
			c.AbortWithError(http.StatusBadGateway, errors.New("error while decoding JWT token"))
		}

		ddb := dbmodule.DynamoClient{
			Client: dynamodb.New(dbmodule.Session),
		}
		attr, err := dynamodbattribute.Marshal(*userEmail)

		if err != nil {
			log.Printf("Error while marshalling email Error: %s", err.Error())

			c.AbortWithError(http.StatusBadGateway, errors.New("error while getting schedules"))
		}

		input := &dynamodb.QueryInput{
			TableName:              &TableName,
			KeyConditionExpression: aws.String("#createdBy = :createdBy"),
			ExpressionAttributeNames: map[string]*string{
				"#createdBy": aws.String("primaryKey"),
			},
			ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
				":createdBy": attr,
			},
		}
		if output, err := ddb.Query(input); err != nil {
			log.Println(err)
			c.AbortWithError(http.StatusBadGateway, errors.New("error while getting schedules"))
		} else {
			var schedules []shared.NewSchedule
			records := make([]shared.Record, 0) // []
			err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &schedules)
			log.Printf("records: %v", records)
			if err != nil {
				log.Println(err)
				c.AbortWithError(http.StatusBadGateway, errors.New("error while getting schedules"))
			}

			for _, schedule := range schedules {

				records = append(records, schedule.ToRecord())
			}

			c.JSON(http.StatusOK, gin.H{
				"records": records,
			})

		}

	})

	schedule.POST("", func(c *gin.Context) {

		log.Println(c.Request.Header)
		token := c.Request.Header["Authorization"][0]
		userEmail, err := getCreatedBy(token)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "error while decoding JWT token",
			})
			return
		}
		new, err := io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error Creating New Schedule",
			})
			return
		}

		var incomingRecord shared.Record
		err = json.Unmarshal(new, &incomingRecord)

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error Creating New Schedule",
			})
			return
		} else {

			incomingRecord.SetCreatedBy(*userEmail)
		}

		schedule, err := incomingRecord.ToNewSchedule()

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error Creating New Schedule",
			})
		}

		item, err := schedule.ToDynamoAttr()
		if err != nil {
			log.Printf("Error converting schedule into dynamoAttr, %v", err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error Creating New Schedule",
			})
			return
		}
		input := dynamodb.PutItemInput{
			TableName: aws.String(TableName),
			Item:      item,
		}

		client := dbmodule.DynamoClient{
			Client: dynamodb.New(dbmodule.Session),
		}

		if _, err = client.PutItem(&input); err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error Creating New Schedule",
			})
			return
		} else {
			c.JSON(http.StatusOK, gin.H{
				"message": "Succesfully Created a schedule",
			})
		}

	})
	schedule.PUT("/:sortKey", func(c *gin.Context) {

		token := c.Request.Header["Authorization"][0]
		userEmail, err := getCreatedBy(token)
		if err != nil {
			log.Println(err)
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "error while decoding JWT token",
			})
			return
		}

		id := c.Params.ByName("sortKey")
		new, err := io.ReadAll(c.Request.Body)
		defer c.Request.Body.Close()

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error while reading body",
			})
			return
		}
		var patchRecord shared.PatchRecord

		err = json.Unmarshal(new, &patchRecord)

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error While Updating Record",
			})
			return
		}
		ExpressionAttributeNames := map[string]*string{}
		ExpressionAttributeValues := map[string]*dynamodb.AttributeValue{}
		UpdateExpression := []string{}

		if patchRecord.DeliveryMethods != nil {
			ExpressionAttributeNames["#deliveryMethods"] = aws.String("deliveryMethods")

			method, err := dynamodbattribute.Marshal(patchRecord.DeliveryMethods)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error while reading body",
				})
				return
			}
			ExpressionAttributeValues[":deliveryMethods"] = method

			UpdateExpression = append(UpdateExpression, "#deliveryMethods = :deliveryMethods")
		}
		if patchRecord.Date != nil {

			timeObj, err := time.Parse(time.RFC3339, *patchRecord.Date)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error while reading body",
				})
				return
			}
			ts := timeObj.Add(time.Hour * 24).Unix()

			ExpressionAttributeNames["#date"] = aws.String("date")
			ExpressionAttributeNames["#TTL"] = aws.String("TTL")
			date, err := dynamodbattribute.Marshal(*patchRecord.Date)
			ttl, ttl_err := dynamodbattribute.Marshal(int(ts))
			if err != nil || ttl_err != nil {
				log.Printf("TTL: %s, Date: %s", ttl_err.Error(), err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error while reading body",
				})
				return
			}
			ExpressionAttributeValues[":date"] = date
			ExpressionAttributeValues[":TTL"] = ttl

			UpdateExpression = append(UpdateExpression, "#date = :date", "#TTL = :TTL")
		}
		UpdateExpressionString := "SET " + strings.Join(UpdateExpression, " , ")
		Key, err := dynamodbattribute.MarshalMap(map[string]string{
			"primaryKey": *userEmail,
			"sortKey":    id,
		})

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error while reading body",
			})
			return
		}

		input := &dynamodb.UpdateItemInput{
			TableName:                 aws.String(TableName),
			Key:                       Key,
			UpdateExpression:          &UpdateExpressionString,
			ExpressionAttributeNames:  ExpressionAttributeNames,
			ExpressionAttributeValues: ExpressionAttributeValues,
			ReturnValues:              aws.String("ALL_NEW"),
		}

		ddb := dbmodule.DynamoClient{
			Client: dynamodb.New(dbmodule.Session),
		}

		output, err := ddb.UpdateItem(input)

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error while reading body",
			})
			return
		}

		updatedItem := output.Attributes
		var newSchedule shared.NewSchedule
		err = dynamodbattribute.UnmarshalMap(updatedItem, &newSchedule)

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error while reading body",
			})
			return
		}
		dto := newSchedule.ToRecord()

		if err != nil {
			log.Println(err.Error())
			c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
				"error": "Error while reading body",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"new": dto,
		})
	})
	if os.Getenv("STAGE") == "local" {
		if err := r.Run(); err != nil {
			log.Fatal("Error while starting the server")
		}
	} else {
		ginLambda = ginadapter.New(r)
	}
}

func Handler(ctx context.Context, req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// If no name is provided in the HTTP request body, throw an error
	return ginLambda.ProxyWithContext(ctx, req)
}
func getCreatedBy(tokenString string) (*string, error) {
	claims := jwt.MapClaims{}
	token := strings.Split(tokenString, " ")[1]
	jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (interface{}, error) {
		return nil, nil
	})
	if email, ok := claims["email"]; !ok {

		return nil, errors.New("error while getting user email from token")

	} else if emailString, ok := email.(string); !ok {

		return nil, errors.New("email is not a string")
	} else {
		return &emailString, nil
	}
}
