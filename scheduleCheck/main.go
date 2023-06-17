package main

import (
	"context"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	if os.Getenv("STAGE") != "local" {
		lambda.Start(handler)
	} else {
		handler(context.Background(), events.CloudWatchEvent{})
	}
}
