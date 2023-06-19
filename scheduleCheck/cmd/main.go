package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	h "github.com/japb1998/lashroom/scheduleCheck/pkg/handler"
)

func main() {
	if os.Getenv("STAGE") != "local" {
		lambda.Start(h.Handler)
	} else {
		h.Handler(context.Background(), events.CloudWatchEvent{})
	}
}
