package main

import (
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/japb1998/lashroom/clientQueue/pkg/handler"
)

func main() {
	lambda.Start(handler.Handler)
}
