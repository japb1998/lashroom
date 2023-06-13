package main

import (
	"os"

	"net/http"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/awslabs/aws-lambda-go-api-proxy/httpadapter"
)

func main() {
	// we init our routes here
	serve()
	if os.Getenv("STAGE") != "local" {
		lambda.Start(httpadapter.New(http.DefaultServeMux).ProxyWithContext)
	}
}
