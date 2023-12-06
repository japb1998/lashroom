package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/scheduler"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/pkg/credentials"
	"github.com/japb1998/control-tower/pkg/email"
	"github.com/joho/godotenv"
)

var notificationSvc *service.NotificationService
var emailSvc *email.EmailService
var clientSvc *service.ClientService

// var msgSvc *sms.MsgSvc
var apiUrl = os.Getenv("API_URL")
var handlerLogger = log.New(os.Stdin, "[Handler] ", log.Default().Flags())

func main() {
	lambda.Start(Handler)
}

func Handler(ctx context.Context, event service.Notification) error {
	var wg sync.WaitGroup
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(event)

	if err != nil {

		for _, ve := range err.(validator.ValidationErrors) {
			fmt.Printf("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
		}
		return fmt.Errorf("error validation ocurred")
	}

	client, err := clientSvc.GetClientById(event.CreatedBy, event.ClientId)
	if *client.OptIn == false {
		fmt.Printf("client User='%s', clientID='%s' has notifications disabled. Skipping.", event.CreatedBy, event.ClientId)
		return nil
	}

	errChan := make(chan error, len(event.DeliveryMethods))
	for _, i := range event.DeliveryMethods {
		switch service.ContactOptions(i) {
		case service.Email:
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, time.Second*30)
				defer cancel()

				if err != nil {
					handlerLogger.Printf("Failed to retrieve client error: %s\n", err)
				}
				tempVars := map[string]any{
					"customer_name": fmt.Sprintf("%s %s", client.FirstName, client.LastName),
					"op_out_url":    fmt.Sprintf("%s/unsubscribe/%s/%s", apiUrl, client.CreatedBy, client.Id),
				}
				message := email.NewEmail("lashroom", "", "2 Weeks Maintenance Reminder", "no-reply@lashroombyeli.me", &tempVars, []string{client.Email}, nil)
				if err := emailSvc.Send(ctx, message); err != nil {
					fmt.Printf("email failed to send error= %s", err)
					errChan <- err

				} else {
					handlerLogger.Printf("Email Sent to='%s'", client.Email)
					errChan <- nil
				}
			}()
		case service.Phone:
			handlerLogger.Println("successfully send message with method='Phone'")
			errChan <- nil
		default:
			handlerLogger.Printf("Invalid delivery method method: %d", i)
			errChan <- fmt.Errorf("Invalid delivery method")
		}
	}
	status := service.SentStatus
	for i := 0; i < len(event.DeliveryMethods); i++ {
		if err := <-errChan; err != nil {
			status = service.FailedStatus
		}
	}
	notificationSvc.SetNotificationStatus(event.CreatedBy, event.ID, status)
	close(errChan)

	wg.Wait()

	return nil
}

func init() {
	var sess *session.Session

	switch os.Getenv("STAGE") {
	case "local":
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}

		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           "personal",
			Config: aws.Config{
				Region: aws.String("us-east-1"),
			},
		}))
	default:
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	}

	var ops email.EmailSvcOpts
	secretArn := os.Getenv("MAIL_GUN_SECRET_ID")
	cm := credentials.NewCredentialsManager(sess)

	b, err := cm.GetSecret(secretArn)

	if err != nil {
		handlerLogger.Fatalf("error getting secret arn='%s', error='%s'", secretArn, err)
	}

	err = json.Unmarshal([]byte(*b), &ops)

	if err != nil {
		handlerLogger.Fatalf("error unmarshalling secret, error='%s'", err)
	}
	// email
	emailSvc = email.NewEmailService(&ops)
	scheduler := scheduler.NewScheduler(sess)

	// message service
	// msgSvc = sms.MusInitMsgSvc(os.Getenv("TWILIO_NUMBER"))

	// notification
	notificationStore := database.NewNotificationRepository(sess)
	notificationSvc = service.NewNotificationService(notificationStore, scheduler)

	// client
	clientStore := database.NewClientRepo(sess)
	clientSvc = service.NewClientSvc(clientStore)
}
