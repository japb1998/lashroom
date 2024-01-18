package controller

import (
	"fmt"
	"log"
	"os"

	"github.com/japb1998/control-tower/internal/apigateway"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/scheduler"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/pkg/awssess"
	"github.com/joho/godotenv"
	"go.opentelemetry.io/otel"
)

var connectionSvc *service.ConnectionSvc
var templateSvc *service.TemplateSvc

func init() {

	if os.Getenv("STAGE") == "local" {

		fmt.Println("init local")
		err := godotenv.Load(".env", "./control-tower/.env")
		if err != nil {
			log.Fatalf("Error loading env vars: %s", err)
		}
	}

	// aws session
	sess := awssess.MustGetSession()

	// scheduler service
	scheduler := scheduler.NewScheduler(sess)
	notificationStore := database.NewNotificationRepository(sess)
	notificationService = service.NewNotificationService(notificationStore, scheduler)
	notificationLogger.Info("initialized scheduler")
	//client service
	clientStore := database.NewClientRepo(sess)
	clientService = service.NewClientSvc(clientStore)
	notificationLogger.Info("initialized client service")
	// ws service
	apigw := apigateway.NewApiGatewayClient(sess, os.Getenv("WS_HTTPS_URL"))
	connStore := database.NewConnectionRepo(sess)
	connectionSvc = service.NewConnectionSvc(connStore, apigw)
	notificationLogger.Info("initialized ws service")

	// template svc
	templateStore := database.NewTemplateRepository(sess)
	templateSvc = service.NewTemplateSvc(templateStore)
	// initialize tracer
	tracer = otel.Tracer("github.com/japb1998/control-tower/internal/controller")
	notificationLogger.Info("Controllers Initialized")
}
