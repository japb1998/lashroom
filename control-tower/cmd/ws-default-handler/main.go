package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/japb1998/control-tower/internal/apigateway"
	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/internal/websocket"
	"github.com/japb1998/control-tower/pkg/awssess"
)

var wsController *websocket.WebSocketController

func main() {

	lambda.StartWithOptions(wsController.HandleDefault)
}

func init() {

	sess := awssess.MustGetSession()

	store := database.NewConnectionRepo(sess)

	apigw := apigateway.NewApiGatewayClient(sess, os.Getenv("WS_HTTPS_URL"))

	service := service.NewConnectionSvc(store, apigw)

	wsController = websocket.NewWSController(service)
}
