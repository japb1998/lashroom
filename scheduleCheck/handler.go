package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/japb1998/lashroom/dbmodule"
	"github.com/japb1998/lashroom/shared"
	"github.com/mailgun/mailgun-go/v3"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

const (
	PHONE shared.ContactOptions = iota
	EMAIL
)

var wg sync.WaitGroup

func handler(_ context.Context, event events.CloudWatchEvent) {
	ddb := dbmodule.DynamoClient{
		Client: dynamodb.New(dbmodule.Session),
	}
	newSchedules, err := GetNotSentNotifications(&ddb)

	if err != nil {
		log.Panicln("Error while getting notifications")
	}
	filterByDeliveryMethod := func(deliveryMethod int8) []shared.NewSchedule {

		emails := make([]shared.NewSchedule, 0)

		for _, ns := range newSchedules {
			if shared.Contains(ns.DeliveryMethods, deliveryMethod) {
				emails = append(emails, ns)
			}
		}
		return emails
	}
	wg.Add(2)
	go sendEmailNotifications(filterByDeliveryMethod(int8(shared.EMAIL)), ddb)
	go TriggerTextNotification(filterByDeliveryMethod(int8(shared.PHONE)), ddb)
	wg.Wait()
}

func sendEmailNotifications(newSchedules []shared.NewSchedule, ddb dbmodule.DynamoClient) {
	defer wg.Done()

	for _, schedule := range newSchedules {
		wg.Add(1)
		go func(schedule shared.NewSchedule) {
			defer wg.Done()
			if _, err := SendEmail(*schedule.Email, 2, "lashroom", map[string]string{
				"customer_name": schedule.ClientName}); err != nil {

				log.Printf("Error While sending Email: %s", err.Error())

				wg.Add(1)
				go func(ddb dbmodule.DynamoClient) {
					defer wg.Done()
					updateNotificationStatus(schedule.PrimaryKey, schedule.SortKey, shared.FAILED, ddb)

				}(ddb)
			} else {
				wg.Add(1)
				go func(ddb dbmodule.DynamoClient) {
					defer wg.Done()
					updateNotificationStatus(schedule.PrimaryKey, schedule.SortKey, shared.SENT, ddb)
				}(ddb)
			}

		}(schedule)
	}

}

func getTodaysHourZero() string {
	// Get the current time in UTC
	currentTime := time.Now().UTC()

	// Set the time to 0 hour (midnight)
	zeroHourTime := time.Date(currentTime.Year(), currentTime.Month(), currentTime.Day(), 0, 0, 0, 0, time.UTC)

	// Format the zero hour time in ISO 8601 format (ISO time)
	isoTime := zeroHourTime.Format(time.RFC3339)

	return isoTime
}

func getTomorrowHourZero() string {
	// Get the current time in UTC
	currentTime := time.Now().UTC()

	// Get tomorrow's date
	tomorrow := currentTime.AddDate(0, 0, 1)

	// Set tomorrow's time to 0 hour (midnight)
	tomorrowMidnight := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)

	// Format tomorrow's midnight time in ISO 8601 format (ISO time)
	isoTime := tomorrowMidnight.Format(time.RFC3339)

	return isoTime
}

func SendEmail(recipient string, weekValue int8, template string, templateVariables map[string]string) (string, error) {

	domain := os.Getenv("EMAIL_DOMAIN")
	apiKey := os.Getenv("EMAIL_API_KEY")

	mg := mailgun.NewMailgun(domain, apiKey)
	m := mg.NewMessage(
		"lashroombyeli@no-reply.com",
		fmt.Sprintf("%d Weeks Reminder", weekValue),
		"Disregarg Message",
		recipient,
	)
	m.SetTemplate(template)

	for k, v := range templateVariables {

		m.AddTemplateVariable(k, v)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	_, id, err := mg.Send(ctx, m)
	return id, err
}
func TriggerTextNotification(newSchedules []shared.NewSchedule, ddb dbmodule.DynamoClient) {

	for _, schedule := range newSchedules {

		wg.Add(1)
		go func(schedule shared.NewSchedule) {
			defer wg.Done()
			if err := SendTextMessage(*schedule.PhoneNumber, schedule.ClientName, fmt.Sprintf(
				"Hi, %s I hope this message finds you feeling as fabulous as ever!"+
					"I just wanted to drop you a quick note to remind you that your 2-week lash maintenance is coming up soon.\n"+
					"LashRoom by Eli\n"+
					"[booksy]\n"+
					"[address]", schedule.ClientName)); err != nil {
				wg.Add(1)
				go func(ddb dbmodule.DynamoClient) {
					defer wg.Done()
					updateNotificationStatus(schedule.PrimaryKey, schedule.SortKey, shared.FAILED, ddb)

				}(ddb)
			} else {
				wg.Add(1)
				go func(ddb dbmodule.DynamoClient) {
					defer wg.Done()
					updateNotificationStatus(schedule.PrimaryKey, schedule.SortKey, shared.SENT, ddb)

				}(ddb)
			}

		}(schedule)

	}

	wg.Done()
}
func SendTextMessage(phoneNumber string, clientName string, text string) error {

	client := twilio.NewRestClient()
	serviceSid := os.Getenv("MESSAGING_SERVICE_SID")
	log.Println(serviceSid)
	params := &api.CreateMessageParams{}
	params.SetBody(text)
	params.SetMessagingServiceSid(serviceSid)
	params.SetTo(phoneNumber)

	resp, err := client.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
		return err
	} else {
		if resp.Sid != nil {
			fmt.Println(*resp.Sid)
		} else {
			fmt.Println(resp.Sid)
		}
	}
	return nil
}
func updateNotificationStatus(pk string, sk string, status string, ddb dbmodule.DynamoClient) error {
	table := os.Getenv("EMAIL_TABLE")
	awsStatus, err := dynamodbattribute.Marshal(status)
	key, keyErr := dynamodbattribute.MarshalMap(map[string]interface{}{
		"primaryKey": pk,
		"sortKey":    sk,
	})
	if err != nil || keyErr != nil {
		log.Println(err, keyErr)
		return err
	}
	input := dynamodb.UpdateItemInput{
		TableName:        &table,
		UpdateExpression: aws.String("SET #status = :status"),
		Key:              key,
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
		},
		ExpressionAttributeValues: map[string]*dynamodb.AttributeValue{
			":status": awsStatus,
		},
	}
	switch status {
	case shared.SENT, shared.FAILED:
		_, err := ddb.UpdateItem(&input)
		return err
	default:
		return errors.New("invalid status when updating notification")
	}
}

func GetNotSentNotifications(ddb *dbmodule.DynamoClient) ([]shared.NewSchedule, error) {
	table := os.Getenv("EMAIL_TABLE")
	startDate := getTodaysHourZero()
	endDate := getTomorrowHourZero()
	attrValue, err := dynamodbattribute.MarshalMap(map[string]interface{}{
		":status":     shared.NOT_SENT,
		":start_date": startDate,
		":end_date":   endDate,
	})
	log.Printf("Any Notifications Between %s and %s\n", startDate, endDate)
	if err != nil {
		log.Println(err)
		return nil, errors.New("error marshalling atttributes")
	}
	input := dynamodb.QueryInput{
		TableName:              &table,
		KeyConditionExpression: aws.String("#status = :status"),
		ExpressionAttributeNames: map[string]*string{
			"#status": aws.String("status"),
			"#date":   aws.String("date"),
		},
		ExpressionAttributeValues: attrValue,
		FilterExpression:          aws.String("#date BETWEEN :start_date AND :end_date"),
		IndexName:                 aws.String("STATUS"),
	}

	if output, err := ddb.Query(&input); err != nil {
		log.Println(err)
		return nil, errors.New("error while querying phone Notifications")
	} else {
		items := output.Items

		var newSchedules []shared.NewSchedule

		err := dynamodbattribute.UnmarshalListOfMaps(items, &newSchedules)

		if err != nil {
			log.Println(err)
			return nil, errors.New("error While Unmarshalling Schedules")
		}
		fmt.Println(newSchedules)
		return newSchedules, nil

	}
}
