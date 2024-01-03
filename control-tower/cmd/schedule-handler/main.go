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
	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/apigateway"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/scheduler"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/pkg/awssess"
	"github.com/japb1998/control-tower/pkg/credentials"
	"github.com/japb1998/control-tower/pkg/email"
	"github.com/japb1998/control-tower/pkg/sms"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda"
	"go.opentelemetry.io/contrib/instrumentation/github.com/aws/aws-lambda-go/otellambda/xrayconfig"
	"go.opentelemetry.io/contrib/propagators/aws/xray"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var notificationSvc *service.NotificationService
var emailSvc *email.EmailService
var clientSvc *service.ClientService
var connectionSvc *service.ConnectionSvc

var msgSvc *sms.MsgSvc
var apiUrl = os.Getenv("API_URL")
var handlerLogger = log.New(os.Stdin, "[Handler] ", log.Default().Flags())
var tracer trace.Tracer

func main() {
	ctx := context.Background()
	tp, err := xrayconfig.NewTracerProvider(ctx)
	if err != nil {
		fmt.Printf("error creating tracer provider: %v", err)
	}
	defer func(ctx context.Context) {
		err := tp.Shutdown(ctx)
		if err != nil {
			fmt.Printf("error shutting down tracer provider: %v", err)
		}
	}(ctx)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(xray.Propagator{})
	tracer = tp.Tracer("main")
	// we init our routes here
	lambda.Start(otellambda.InstrumentHandler(Handler, xrayconfig.WithRecommendedOptions(tp)...))
}

func Handler(ctx context.Context, event service.Notification) error {
	c, span := tracer.Start(ctx, "handler-start")
	defer span.End()

	var wg sync.WaitGroup
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(event)

	if err != nil {

		for _, ve := range err.(validator.ValidationErrors) {
			fmt.Printf("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
		}
		return fmt.Errorf("error validation ocurred")
	}

	// get client span
	_, getClientSpan := tracer.Start(c, "get-client")
	client, err := clientSvc.GetClientById(event.CreatedBy, event.ClientId)

	if err != nil {
		getClientSpan.End()
		return err
	}
	if *client.OptIn == false {
		fmt.Printf("client User='%s', clientID='%s' has notifications disabled. Skipping.", event.CreatedBy, event.ClientId)
		getClientSpan.End()
		return nil
	}
	getClientSpan.End()

	_, deliverySpan := tracer.Start(c, "delivery-methods-loop")
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
				message := email.NewEmail("lashroom", "", "Lash Room - 2 Weeks Maintenance Reminder", "no-reply@lashroombyeli.me", &tempVars, []string{client.Email}, nil)
				if err := emailSvc.Send(ctx, message); err != nil {
					fmt.Printf("email failed to send error= %s\n", err)
					errChan <- err

				} else {
					handlerLogger.Printf("Email Sent to='%s'\n", client.Email)
					errChan <- nil
				}
			}()
		case service.Phone:
			wg.Add(1)
			go func() {
				defer wg.Done()
				templateVariables, err := json.Marshal(map[int]string{
					1: client.FirstName,
					2: "2",
				})
				if err != nil {
					errChan <- err
					return
				}

				msg := sms.Msg{
					To:                client.Phone,
					TemplateVariables: templateVariables,
					TemplateId:        os.Getenv("TWILIO_TEMPLATE_ID"),
				}

				if err := msgSvc.SendMessage(&msg); err != nil {
					handlerLogger.Printf("failed to send message %s\n", err)
					errChan <- err

				} else {
					handlerLogger.Printf("SMS was successfully sent to='%s'!\n", client.Phone)
					errChan <- nil
				}

			}()

		default:
			handlerLogger.Printf("Invalid delivery method method: %d\n", i)
			errChan <- fmt.Errorf("Invalid delivery method")
		}
	}

	status := service.SentStatus
	for i := 0; i < len(event.DeliveryMethods); i++ {
		if err := <-errChan; err != nil {
			status = service.FailedStatus
		}
	}
	if status == service.FailedStatus {
		deliverySpan.SetStatus(codes.Error, err.Error())
	}
	// end delivery span
	deliverySpan.End()

	_, statusSpan := tracer.Start(c, "set-status-span")
	err = notificationSvc.SetNotificationStatus(event.CreatedBy, event.ID, status)

	if err != nil {
		statusSpan.SetStatus(codes.Error, err.Error())
		statusSpan.End()
		return err
	}
	statusSpan.End()
	close(errChan)

	// we are done waiting for the go rutine to finish
	wg.Wait()

	// notify active connections - FE.
	wsCtx, wsSpan := tracer.Start(c, "ws-span")
	msg, err := service.NewNotificationUpdateMsg(client.CreatedBy, event.ID).WithAction(service.NotificationUpdatedAction)

	// we do not return an error because an error here does not mean that the delivery failed
	if err != nil {
		handlerLogger.Println(err)
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		handlerLogger.Println(err)
		wsSpan.End()
	}
	handlerLogger.Printf("WS message='%v'", msg)

	err = connectionSvc.SendWsMessageByEmail(wsCtx, msg)
	// we do not return an error because an error here does not mean that the delivery failed
	if err != nil {
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		handlerLogger.Println(err)
		wsSpan.End()
	}
	wsSpan.End()

	return nil
}

func init() {
	// if local load from .env file
	switch os.Getenv("STAGE") {
	case "local":
		err := godotenv.Load(".env")
		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}
	}

	sess := awssess.MustGetSession()

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
	msgSvc = sms.MusInitMsgSvc(os.Getenv("TWILIO_SERVICE_ID"))

	// notification
	notificationStore := database.NewNotificationRepository(sess)
	notificationSvc = service.NewNotificationService(notificationStore, scheduler)

	// client
	clientStore := database.NewClientRepo(sess)
	clientSvc = service.NewClientSvc(clientStore)

	// connection
	connRepo := database.NewConnectionRepo(sess)
	apigw := apigateway.NewApiGatewayClient(sess, os.Getenv("WS_HTTPS_URL"))
	connectionSvc = service.NewConnectionSvc(connRepo, apigw)
}
