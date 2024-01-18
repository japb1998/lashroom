package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/japb1998/control-tower/internal/service"
	"github.com/japb1998/control-tower/pkg/email"
)

var notificationSvc *service.NotificationService
var emailSvc *email.EmailService
var clientSvc *service.ClientService
var connectionSvc *service.ConnectionSvc

func sendEmailReminder(ctx context.Context, to []string, payload map[string]any) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	message := email.NewEmail("lashroom", "", fmt.Sprintf("Lash Room - %s Weeks Maintenance Reminder", payload["weeks"].(string)), "no-reply@lashroombyeli.me", &payload, to, nil)
	if err := emailSvc.Send(ctx, message); err != nil {
		fmt.Printf("email failed to send error= %s\n", err)
		return err

	} else {
		handlerLogger.Info("Email Sent", slog.Any("to", to), slog.Any("payload", payload))
		return err
	}
}
