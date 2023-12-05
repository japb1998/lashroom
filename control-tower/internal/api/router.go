package api

import (
	"log"
	"os"
	"time"

	"github.com/aws/aws-lambda-go/events"
	ginadapter "github.com/awslabs/aws-lambda-go-api-proxy/gin"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	docs "github.com/japb1998/control-tower/docs"
	"github.com/japb1998/control-tower/internal/controller"
	swaggerfiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type Response events.APIGatewayProxyResponse

var ginLambda *ginadapter.GinLambda

var (
	TableName    = os.Getenv("EMAIL_TABLE")
	ClientTable  = os.Getenv("CLIENT_TABLE")
	queueUrl     = os.Getenv("QUEUE_URL")
	routerLogger = log.New(os.Stdout, "[Router] ", log.Default().Flags())
)

func Serve() {
	routerLogger.Printf("Gin cold start")
	r := gin.Default()
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("rfc3339", func(fl validator.FieldLevel) bool {
			routerLogger.Println("registering validation rfc3339")
			if fl.Field().String() == "" {
				return true
			}

			_, err := time.Parse(time.RFC3339, fl.Field().String())

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

	if os.Getenv("STAGE") == "local" {
		if err := r.Run(os.Getenv("PORT")); err != nil {
			routerLogger.Fatal("Error while starting the server")
		}
	} else {
		ginLambda = ginadapter.New(r)
	}

}
