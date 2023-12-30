package websocket

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/japb1998/control-tower/internal/service"
)

var (
	wsLogger = log.New(os.Stdin, "[WebSocket Controller] ", log.Default().Flags())
)

// ConnectionSvc
type ConnectionSvc interface {
	SendUpdateByEmail(ctx context.Context, msg *service.NotificationUpdate) error
	Connect(ctx context.Context, conn *service.Connection) error
	Disconnect(ctx context.Context, conn *service.Connection) (err error)
}

// WebSocketController
type WebSocketController struct {
	svc ConnectionSvc
}

// NewWSController returns a pointer to a ws controller
func NewWSController(svc ConnectionSvc) *WebSocketController {
	return &WebSocketController{
		svc,
	}
}

// HandleConnection -  handles connection routes for AWS apigateway websocket API
func (c *WebSocketController) HandleConnection(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {
	routeKey := event.RequestContext.RouteKey
	switch routeKey {
	case "$connect":
		{
			from, err := getEmailFromContext(event.RequestContext.Authorizer)

			if err != nil {
				return events.APIGatewayProxyResponse{}, err
			}
			messageUrl := fmt.Sprintf("https://%s/%s", event.RequestContext.DomainName, event.RequestContext.Stage)
			cId := event.RequestContext.ConnectionID

			wsLogger.Printf("url=%s, from=%s, cId=%s\n", messageUrl, from, cId)
			conn := &service.Connection{
				Email:        from,
				ConnectionId: cId,
			}
			if err := c.svc.Connect(ctx, conn); err != nil {
				return events.APIGatewayProxyResponse{}, err
			}

			wsLogger.Println("Successfully connected!")
			return events.APIGatewayProxyResponse{
				StatusCode:      200,
				Headers:         nil,
				IsBase64Encoded: false,
			}, nil
		}
	case "$disconnect":
		{
			from, err := getEmailFromContext(event.RequestContext.Authorizer)

			if err != nil {
				return events.APIGatewayProxyResponse{}, err
			}
			messageUrl := fmt.Sprintf("https://%s/%s", event.RequestContext.DomainName, event.RequestContext.Stage)
			cId := event.RequestContext.ConnectionID

			fmt.Printf("url=%s, from=%s, cId=%s\n", messageUrl, from, cId)
			conn := &service.Connection{
				Email:        from,
				ConnectionId: cId,
			}
			if err := c.svc.Disconnect(ctx, conn); err != nil {
				return events.APIGatewayProxyResponse{}, err
			}
			wsLogger.Println("Successfully disconnected!")
			return events.APIGatewayProxyResponse{
				StatusCode:      200,
				Headers:         nil,
				IsBase64Encoded: false,
			}, nil
		}

	}

	return events.APIGatewayProxyResponse{
		StatusCode:      404,
		Headers:         nil,
		IsBase64Encoded: false,
	}, nil
}

// HandleDefault - TBD
func (c *WebSocketController) HandleDefault(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayProxyResponse, error) {

	msgUrl := fmt.Sprintf("https://%s/%s", event.RequestContext.DomainName, event.RequestContext.Stage)
	cId := event.RequestContext.ConnectionID

	wsLogger.Printf("url=%s, cId=%s\n", msgUrl, cId)

	wsLogger.Printf("received message='%s'\n", event.Body)

	email, err := getEmailFromContext(event.RequestContext.Authorizer)

	if err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	msg := &service.NotificationUpdate{
		Email:          email,
		NotificationId: "sample_id",
	}

	if err := c.svc.SendUpdateByEmail(ctx, msg); err != nil {
		return events.APIGatewayProxyResponse{}, err
	}

	return events.APIGatewayProxyResponse{
		StatusCode:      200,
		Headers:         nil,
		IsBase64Encoded: false,
	}, nil
}