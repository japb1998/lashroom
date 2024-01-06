package service

import (
	"os"

	"log/slog"
)

// Client Logger
var (
	clientHandler = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "Client Service")})
	clientLogger  = slog.New(clientHandler)
)

// Notification Logger
var (
	nLoggerHandler     = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "Notification Service")})
	notificationLogger = slog.New(nLoggerHandler)
)

// Connection Logger
var (
	connectionHandler = slog.NewTextHandler(os.Stdout, nil).WithAttrs([]slog.Attr{slog.String("name", "Connection Service")})
	connectionLogger  = slog.New(connectionHandler)
)
