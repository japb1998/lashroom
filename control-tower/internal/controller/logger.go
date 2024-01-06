package controller

import (
	"log/slog"
	"os"
)

// handlers
var (
	clientHandler       = slog.NewTextHandler(os.Stdin, nil).WithAttrs([]slog.Attr{slog.String("name", "clientController")})
	notificationHandler = slog.NewTextHandler(os.Stdin, nil).WithAttrs([]slog.Attr{slog.String("name", "notificationController")})
)

// loggers
var (
	clientLogger       = slog.New(clientHandler)
	notificationLogger = slog.New(notificationHandler)
)
