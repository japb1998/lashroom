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

func sendEmailReminder(ctx context.Context, firstName, lastName, opOutUrl string, to []string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	tempVars := map[string]any{
		"customer_name": fmt.Sprintf("%s %s", firstName, lastName),
		"op_out_url":    opOutUrl,
	}
	message := email.NewEmail("lashroom", "", "Lash Room - 2 Weeks Maintenance Reminder", "no-reply@lashroombyeli.me", &tempVars, to, nil)
	if err := emailSvc.Send(ctx, message); err != nil {
		fmt.Printf("email failed to send error= %s\n", err)
		return err

	} else {
		handlerLogger.Info("Email Sent", slog.Any("to", to))
		return err
	}
}
