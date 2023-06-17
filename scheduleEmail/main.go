package main

import (
	"os"

	"github.com/aws/aws-lambda-go/lambda"
)

func main() {
	// we init our routes here
	serve()
	if os.Getenv("STAGE") != "local" {
		lambda.Start(Handler)
	}
}
