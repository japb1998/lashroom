package controller

import (
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/scheduler"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
)

func init() {
	// aws session
	var sess *session.Session

	switch os.Getenv("STAGE") {
	case "local":
		fmt.Println("init local")
		err := godotenv.Load(".env", "./control-tower/.env")
		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}
		fmt.Println("local", os.Getenv("EMAIL_TABLE"))
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

	scheduler := scheduler.NewScheduler(sess)
	notificationStore := database.NewNotificationRepository(sess)
	notificationService = service.NewNotificationService(notificationStore, scheduler)
	//client service
	clientStore := database.NewClientRepo(sess)
	clientService = service.NewClientSvc(clientStore)
	notificationLogger.Println("Controllers Initialized")

	// initialize tracer
	tracer = otel.Tracer("github.com/japb1998/control-tower/internal/controller")
}
