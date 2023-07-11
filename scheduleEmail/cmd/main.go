package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/japb1998/lashroom/scheduleEmail/pkg/api"
)

func main() {
	// we init our routes here
	api.Serve()
	if os.Getenv("STAGE") != "local" {
		lambda.Start(api.Handler)
	}
}