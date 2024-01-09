package main

import (
	"encoding/json"
	"log"
	"log/slog"
	"os"

	"github.com/japb1998/control-tower/internal/apigateway"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/scheduler"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/pkg/awssess"
	"github.com/japb1998/control-tower/pkg/credentials"
	"github.com/japb1998/control-tower/pkg/email"
	"github.com/japb1998/control-tower/pkg/sms"
	"github.com/joho/godotenv"
)

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
		handlerLogger.Error("error getting secret", slog.String("arn", secretArn), slog.String("error", err.Error()))
		panic("error initializing handler")
	}

	err = json.Unmarshal([]byte(*b), &ops)

	if err != nil {
		handlerLogger.Error("error unmarshalling secret", slog.String("error", err.Error()))
		panic("error initializing handler")
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
