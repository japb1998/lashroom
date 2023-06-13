package main

import "github.com/aws/aws-lambda-go/lambda"

func main() {
	lambda.Start(handler)
	// handler(context.Background(), events.CloudWatchEvent{})
}
