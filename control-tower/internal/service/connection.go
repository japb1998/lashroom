package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
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

// ws-actions
const (
	NotificationUpdatedAction = "updateNotification"  // properties of an specific notification changed.
	NotificationCreatedAction = "newNotification"     // new notification was created.
	NotificationDeletedAction = "notificationDeleted" // a notification was deleted.
	PingResponseAction        = "health-response"
	PingAction                = "health" // ping action to keep the connection alive for longer than 10mins.
)

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

// webSocketMsg
type webSocketMsg struct {
	/**
	Valid Actions
	notificationUpdated = "updateNotifications" // properties of an specific notification changed.
	notificationCreated = "newNotification" // new notification was created.
	notificationDeleted = "notificationDeleted" // a notification was deleted.
	*/
	Action string `json:"action"`
}

type NotificationUpdate struct {
	webSocketMsg
	Email          string `json:"email"`
	NotificationId string `json:"notificationId"`
}

func NewNotificationUpdateMsg(email, notificationId string) *NotificationUpdate {
	return &NotificationUpdate{
		Email:          email,
		NotificationId: notificationId,
	}
}

// WithAction - receives an action string and returns the *NotificationUpdate pointer or error if the action is invalid.
func (n *NotificationUpdate) WithAction(action string) (*NotificationUpdate, error) {
	switch action {
	case NotificationUpdatedAction:
		fallthrough
	case NotificationCreatedAction:
		fallthrough
	case NotificationDeletedAction:
		n.Action = action
		return n, nil
	}

	return nil, fmt.Errorf("invalid action for notification update action='%s'", action)
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

// SendWsMessageByEmail sends a notification update message to all active connections for a user email.
func (c *ConnectionSvc) SendWsMessageByEmail(ctx context.Context, msg *NotificationUpdate) error {

	if msg.Action == "" {
		return fmt.Errorf("message action can't be empty")
	}

	var wg sync.WaitGroup
	conns, err := c.store.GetConnectionIds(ctx, msg.Email)

	if err != nil {
		connectionLogger.Error(err.Error())
		return fmt.Errorf("error getting all active connections for email='%s'", msg.Email)
	}

	d, err := json.Marshal(msg)

	if err != nil {
		connectionLogger.Error(err.Error())
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
				connectionLogger.Error("failed to send message,", slog.String("connectionID", conn.ConnectionId), slog.String("email", conn.Email))
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
		connectionLogger.Error(err.Error())
		return fmt.Errorf("failed to connect connectionId='%s', client='%s'", conn.ConnectionId, conn.Email)
	}

	connectionLogger.Error("Successfully connected!.", slog.String("connectionId", conn.ConnectionId), slog.String("client", conn.Email))
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

// Ping - response to health action
func (c *ConnectionSvc) Ping(ctx context.Context, conn *Connection) error {

	d, err := json.Marshal(webSocketMsg{
		Action: PingResponseAction,
	})

	if err != nil {
		connectionLogger.Error(err.Error())
		return err
	}
	if _, err := c.broadCastClient.PostToConnection(&apigatewaymanagementapi.PostToConnectionInput{
		ConnectionId: &conn.ConnectionId,
		Data:         d,
	}); err != nil {
		connectionLogger.Error("failed to send message,", slog.String("connectionID", conn.ConnectionId), slog.String("email", conn.Email))
	}
	return nil
}
