package api

import (
	"encoding/json"
	"errors"
	"fmt"
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
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/japb1998/lashroom/clientQueue/pkg/operations"
	sqsRecord "github.com/japb1998/lashroom/clientQueue/pkg/record"
	"github.com/japb1998/lashroom/scheduleEmail/pkg/client"
	"github.com/japb1998/lashroom/scheduleEmail/pkg/notification"
	"github.com/japb1998/lashroom/shared/pkg/database"
	"github.com/japb1998/lashroom/shared/pkg/record"
)

type Response events.APIGatewayProxyResponse

var ginLambda *ginadapter.GinLambda

var (
	TableName   = os.Getenv("EMAIL_TABLE")
	ClientTable = os.Getenv("CLIENT_TABLE")
	queueUrl    = os.Getenv("QUEUE_URL")
)

func Serve() {
	// stdout and stderr are sent to AWS CloudWatch Logs
	log.Printf("Gin cold start")
	r := gin.Default()

	corsConfig := cors.DefaultConfig()

	corsConfig.AllowOrigins = []string{"*"}

	// To be able to send tokens to the server.
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AddAllowMethods("OPTIONS", "GET", "PUT", "PATCH")
	r.Use(cors.New(corsConfig))
	r.Use(currentUserMiddleWare())

	// NOTIFICATIONS ROUTER
	schedule := r.Group("/schedule")
	{
		schedule.GET("", func(c *gin.Context) {
			userEmail := c.MustGet("email").(string)
			ddb := database.DynamoClient{
				Client: dynamodb.New(database.Session),
			}
			attr, err := dynamodbattribute.Marshal(userEmail)

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
				var schedules []record.NewSchedule
				records := make([]record.Record, 0) // []
				err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &schedules)

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

			userEmail := c.MustGet("email").(string)

			new, err := io.ReadAll(c.Request.Body)
			defer c.Request.Body.Close()

			if err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error Creating New Schedule",
				})
				return
			}

			var incomingRecord record.Record
			err = json.Unmarshal(new, &incomingRecord)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error Creating New Schedule",
				})
				return
			} else {

				incomingRecord.SetCreatedBy(userEmail)
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

			client := database.DynamoClient{
				Client: dynamodb.New(database.Session),
			}

			if _, err = client.PutItem(&input); err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error Creating New Schedule",
				})
				return
			} else {

				if sqsClient, err := operations.NewSQSClient(nil, &queueUrl); err != nil {
					log.Printf("error while creating sqs client for client creation, %v", err.Error())
				} else {
					clientBody, err := incomingRecord.ToClient()

					if err == nil {
						message := sqsRecord.Event{
							Type: "client",
							Body: clientBody,
						}
						if body, err := json.Marshal(message); err != nil {
							log.Println(err.Error())
						} else {
							sqsClient.SendMessage(body)
						}

					} else {
						log.Println(err.Error())
					}
				}

				c.JSON(http.StatusOK, gin.H{
					"message": "Succesfully Created a schedule",
				})
			}

		})
		schedule.PUT("/:sortKey", func(c *gin.Context) {

			userEmail := c.MustGet("email").(string)

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
			var patchRecord record.PatchRecord

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
				"primaryKey": userEmail,
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

			ddb := database.NewDynamoClient()

			output, err := ddb.UpdateItem(input)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "Error while reading body",
				})
				return
			}

			updatedItem := output.Attributes
			var newSchedule record.NewSchedule
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
		schedule.DELETE("/:id", func(c *gin.Context) {

			userEmail := c.MustGet("email").(string)
			notificationId, _ := c.Params.Get("id")

			client := database.NewDynamoClient()
			notificationStore := database.NotificationRepository{
				Client: client,
			}

			notificationService := notification.NewNotificationService(&notificationStore)

			err := notificationService.DeleteNotification(userEmail, notificationId)

			if err != nil {
				log.Println(err.Error())

				c.AbortWithError(http.StatusBadGateway, fmt.Errorf("error while deleting notification"))

				return
			}

			c.AbortWithStatus(http.StatusNoContent)
		})
	}

	//CLIENT ROUTER
	clients := r.Group("/clients")
	{
		clients.GET("", func(c *gin.Context) {

			userEmail := c.MustGet("email").(string)
			store := database.NewClientRepository()
			clientService := client.NewClientService(store)

			if clientDtoList, err := clientService.GetClientsByCreator(userEmail); err != nil {

				log.Println(err.Error())
				c.AbortWithStatusJSON(http.StatusBadGateway, map[string]string{
					"message": "Error while retreiving clients",
				})

			} else {

				c.JSON(http.StatusOK, gin.H{
					"records": clientDtoList,
					"count":   len(clientDtoList),
				})
			}
		})

		clients.POST("", func(c *gin.Context) {
			userEmail := c.MustGet("email").(string)
			var clientDto client.ClientDto
			store := database.NewClientRepository()
			clientService := client.NewClientService(store)
			body, err := io.ReadAll(c.Request.Body)

			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadRequest, "invalid Body")
				return
			}

			err = json.Unmarshal(body, &clientDto)
			clientDto.CreatedBy = userEmail

			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "invalid json body",
				})
			}

			if client, err := clientService.CreateClient(clientDto); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "error updating user",
				})
			} else {
				c.JSON(http.StatusOK, client)
			}

		})
		clients.PATCH("/:id", func(c *gin.Context) {
			userEmail := c.MustGet("email").(string)
			clientId, _ := c.Params.Get("id")
			var clientDto client.ClientDto
			store := database.NewClientRepository()
			clientService := client.NewClientService(store)
			body, err := io.ReadAll(c.Request.Body)

			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadRequest, "invalid Body")
				return
			}

			err = json.Unmarshal(body, &clientDto)
			if err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
					"error": "invalid json body",
				})
			}

			if client, err := clientService.UpdateUser(userEmail, clientId, clientDto); err != nil {
				log.Println(err)
				c.AbortWithStatusJSON(http.StatusBadGateway, gin.H{
					"error": "error updating user",
				})
			} else {
				c.JSON(http.StatusOK, client)
			}
		})
		clients.GET("/:id", func(c *gin.Context) {
			userEmail := c.MustGet("email").(string)
			clientId, _ := c.Params.Get("id")

			store := database.NewClientRepository()
			clientService := client.NewClientService(store)

			client, err := clientService.GetClientById(userEmail, clientId)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithError(http.StatusBadGateway, errors.New("error while deleting user"))
				return
			} else if client == nil {
				c.AbortWithStatus(http.StatusNotFound)
				return
			}

			c.JSON(http.StatusOK, client)
		})
		clients.DELETE("/:id", func(c *gin.Context) {
			userEmail := c.MustGet("email").(string)
			clientId, _ := c.Params.Get("id")

			store := database.NewClientRepository()
			clientService := client.NewClientService(store)

			err := clientService.DeleteClient(userEmail, clientId)

			if err != nil {
				log.Println(err.Error())
				c.AbortWithError(http.StatusBadGateway, errors.New("error while deleting user"))
				return
			}

			c.AbortWithStatus(http.StatusNoContent)
		})
	}

	if os.Getenv("STAGE") == "local" {
		if err := r.Run(os.Getenv("PORT")); err != nil {
			log.Fatal("Error while starting the server")
		}
	} else {
		ginLambda = ginadapter.New(r)
	}
}
