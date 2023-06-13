package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/japb1998/eliemail/dbmodule"
	"github.com/japb1998/eliemail/shared"
)

type Response events.APIGatewayProxyResponse

var (
	TableName = os.Getenv("EMAIL_TABLE")
)

func serve() {

	http.HandleFunc("/schedule", func(w http.ResponseWriter, r *http.Request) {
		// Read body and defer to close at the end of the function
		switch r.Method {
		case "POST":
			{
				new, err := io.ReadAll(r.Body)
				defer r.Body.Close()

				if err != nil {
					log.Println(err)
					http.Error(w, "Error while reading body", http.StatusBadGateway)
					return
				}

				var incomingRecord shared.Record
				err = json.Unmarshal(new, &incomingRecord)

				if err != nil {
					log.Println(err)
					http.Error(w, "Error while unmarshalling coming request", http.StatusBadGateway)
					return
				}
				schedule, err := incomingRecord.ToNewSchedule()

				if err != nil {
					log.Println(err)
					http.Error(w, fmt.Sprintf("Error converting incoming request into a schedule: %s", err), http.StatusBadGateway)
					return
				}

				item, err := schedule.ToDynamoAttr()
				if err != nil {
					log.Println(err)
					http.Error(w, "Error converting schedule into dynamoAttr", http.StatusBadGateway)
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
					log.Println(err)
					http.Error(w, "Error while adding the item to the database", http.StatusBadGateway)
				} else {
					successResponse, _ := json.Marshal(map[string]string{
						"message": "Succesfully Created a schedule",
					})

					w.Header().Add("content-type", "application/json")
					w.Write(successResponse)
				}
			}
		case "GET":
			{
				ddb := dbmodule.DynamoClient{
					Client: dynamodb.New(dbmodule.Session),
				}
				input := &dynamodb.ScanInput{
					TableName: &TableName,
				}
				if output, err := ddb.Scan(input); err != nil {
					log.Println(err)
					http.Error(w, "error while getting schedules", http.StatusBadGateway)
				} else {
					var schedules []shared.NewSchedule
					records := make([]shared.Record, 0) // []
					err = dynamodbattribute.UnmarshalListOfMaps(output.Items, &schedules)

					if err != nil {
						log.Println(err)
						http.Error(w, "error while getting schedules", http.StatusBadGateway)
					}

					for _, schedule := range schedules {

						records = append(records, schedule.ToRecord())
					}

					response := map[string][]shared.Record{
						"records": records,
					}

					if jsonResponse, err := json.Marshal(response); err != nil {
						log.Println(err)
						http.Error(w, "error while getting schedules", http.StatusBadGateway)
					} else {
						w.Header().Add("content-type", "application/json")
						w.Write(jsonResponse)
					}

				}

			}
		default:
			{
				http.Error(w, "Not Found", http.StatusNotFound)
			}
		}
	})

	if os.Getenv("STAGE") == "local" {
		if err := http.ListenAndServe(":3000", nil); err != nil {
			log.Fatal("Error while starting the server")
		}
	}
}
