package apigateway

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/apigatewaymanagementapi"
)

func NewApiGatewayClient(sess *session.Session, domain string) *apigatewaymanagementapi.ApiGatewayManagementApi {

	client := apigatewaymanagementapi.New(sess, &aws.Config{
		Endpoint: aws.String(domain),
	})

	return client
}
