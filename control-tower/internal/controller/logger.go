package controller

import (
	"log/slog"
	"os"
)

// handlers
var (
	clientHandler       = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "client-controller")})
	notificationHandler = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "notification-controller")})
	templateHandler     = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "template-controller")})
)

// loggers
var (
	clientLogger       = slog.New(clientHandler)
	notificationLogger = slog.New(notificationHandler)
	templateLogger     = slog.New(templateHandler)
)
