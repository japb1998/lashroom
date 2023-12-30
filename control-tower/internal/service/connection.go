package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
	"github.com/japb1998/control-tower/internal/model"
)

// const variables
const (
	enterChat          = "enter"
	leaveChat          = "leave"
	updateNotification = "updateNotification"
)

var connectionLogger = log.New(os.Stdin, "[Connection Service] ", log.Default().Flags())

type ConnectionSvc struct {
	store           ConnectionRepo
	broadCastClient BroadcastClient
}

type ConnectionRepo interface {
	GetConnectionIds(ctx context.Context, email string) ([]model.Connection, error)
	DeleteConnection(ctx context.Context, conn model.Connection) error
	SaveConnection(ctx context.Context, conn model.Connection) error
}

type BroadcastClient interface {
	PostToConnection(*apigatewaymanagementapi.PostToConnectionInput) (*apigatewaymanagementapi.PostToConnectionOutput, error)
}

type webSocketMsg struct {
	Action string `json:"action"`
}

type NotificationUpdate struct {
	webSocketMsg
	Email          string `json:"email"`
	NotificationId string `json:"notificationId"`
}

type Connection struct {
	Email        string `json:"email"`
	ConnectionId string `json:"connectionId"`
}

func NewConnectionSvc(store ConnectionRepo, bClient BroadcastClient) *ConnectionSvc {
	return &ConnectionSvc{
		store:           store,
		broadCastClient: bClient,
	}
}

// SendUpdateByEmail sends a notification update message to all active connections for a user email.
func (c *ConnectionSvc) SendUpdateByEmail(ctx context.Context, msg *NotificationUpdate) error {
	// set the action for routing purposes
	msg.Action = updateNotification

	var wg sync.WaitGroup
	conns, err := c.store.GetConnectionIds(ctx, msg.Email)

	if err != nil {
		connectionLogger.Println(err)
		return fmt.Errorf("error getting all active connections for email='%s'", msg.Email)
	}

	d, err := json.Marshal(msg)

	if err != nil {
		connectionLogger.Println(err)
		return fmt.Errorf("invalid notification message")
	}
	for _, conn := range conns {
		wg.Add(1)

		go func(conn model.Connection) {
			defer wg.Done()

			if _, err := c.broadCastClient.PostToConnection(&apigatewaymanagementapi.PostToConnectionInput{
				ConnectionId: &conn.ConnectionId,
				Data:         d,
			}); err != nil {
				connectionLogger.Printf("failed to send message to connectionID='%s' client='%s'", conn.ConnectionId, conn.Email)
			}

		}(conn)
	}
	wg.Wait()
	return nil
}

// Connect
func (c *ConnectionSvc) Connect(ctx context.Context, conn *Connection) error {
	connection := model.Connection{
		ConnectionId: conn.ConnectionId,
		Email:        conn.Email,
	}
	err := c.store.SaveConnection(ctx, connection)

	if err != nil {
		connectionLogger.Println(err)
		return fmt.Errorf("failed to connect connectionId='%s', client='%s'", conn.ConnectionId, conn.Email)
	}

	connectionLogger.Printf("Successfully connected!. connectionId='%s', client='%s'", conn.ConnectionId, conn.Email)
	return nil
}

// Disconnect
func (c *ConnectionSvc) Disconnect(ctx context.Context, conn *Connection) (err error) {
	connection := model.Connection{
		Email:        conn.Email,
		ConnectionId: conn.ConnectionId,
	}

	err = c.store.DeleteConnection(ctx, connection)

	return err
}
