package api

import (
	"log"
	"os"
	"reflect"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	docs "github.com/japb1998/control-tower/docs"
	"github.com/japb1998/control-tower/internal/controller"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

type Response events.APIGatewayProxyResponse

var (
	TableName    = os.Getenv("EMAIL_TABLE")
	ClientTable  = os.Getenv("CLIENT_TABLE")
	queueUrl     = os.Getenv("QUEUE_URL")
	routerLogger = log.New(os.Stdout, "[Router] ", log.Default().Flags())
)

const (
	ScopeName = "github.com/japb1998/control-tower/internal/api"
)

func InitRoutes() *gin.Engine {
	routerLogger.Printf("Gin cold start")
	r := gin.Default()
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("rfc3339", func(fl validator.FieldLevel) bool {

			var field string
			if reflect.PointerTo(fl.Field().Type()).Kind() == reflect.String {
				if !fl.Field().Addr().IsNil() {
					return true
				}
				field = fl.Field().Addr().String()
			} else {
				if fl.Field().String() == "" {
					return true
				}
				field = fl.Field().String()
			}

			_, err := time.Parse(time.RFC3339, field)

			if err != nil {
				return false
			}

			return true
		})
	}
	corsConfig := cors.DefaultConfig()

	corsConfig.AllowOrigins = []string{"*"}

	// To be able to send tokens to the server.
	corsConfig.AllowCredentials = true
	corsConfig.AllowHeaders = []string{"*"}
	corsConfig.AddAllowMethods("OPTIONS", "GET", "PUT", "PATCH")

	r.Use(otelgin.Middleware(ScopeName))

	r.Use(cors.New(corsConfig))

	// SWAGGER
	docs.SwaggerInfo.BasePath = ""
	{
		r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerfiles.Handler))
	}
	// Email Op-out
	unsubscribe := r.Group("/unsubscribe")
	{
		unsubscribe.GET("/:creator/:userID", controller.OptOut)
	}
	r.Use(currentUserMiddleWare())

	// NOTIFICATIONS ROUTER
	schedule := r.Group("/schedule")
	{
		schedule.GET("", controller.GetSchedules)
		schedule.POST("", controller.PostSchedule)
		schedule.GET("/:id", controller.GetSchedule)
		schedule.PATCH("/:id", controller.UpdateSchedule)
		schedule.DELETE("/:id", controller.DeleteSchedule)
	}

	//CLIENT ROUTER
	clients := r.Group("/clients")
	{
		clients.GET("", controller.ClientsWithFilters)
		clients.POST("", controller.CreateClient)
		clients.PATCH("/:id", controller.UpdateClient)
		clients.GET("/:id", controller.GetClientByID)
		clients.DELETE("/:id", controller.DeleteClient)
	}

	return r

}
