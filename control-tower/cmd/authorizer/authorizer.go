package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/MicahParks/keyfunc/v2"
	"github.com/aws/aws-lambda-go/events"
	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("Invalid token")
	regionID        = os.Getenv("AWS_REGION")
	accountId       = os.Getenv("AWS_ACCOUNT_ID")
	apiId           = os.Getenv("API_ID")
)

// isValid verifies if the JWT token is valid
func isValid(t string) (jwt.MapClaims, error) {
	// Get the JWK Set URL from your AWS region and userPoolId.
	//
	// See the AWS docs here:
	// https://docs.aws.amazon.com/cognito/latest/developerguide/amazon-cognito-user-pools-using-tokens-verifying-a-jwt.html
	userPoolID := os.Getenv("USER_POOL")
	jwksURL := fmt.Sprintf("https://cognito-idp.%s.amazonaws.com/%s/.well-known/jwks.json", regionID, userPoolID)

	// Create the keyfunc.Keyfunc.
	jwks, err := keyfunc.Get(jwksURL, keyfunc.Options{})
	if err != nil {
		log.Fatalf("Failed to create JWK Set from resource at the given URL.\nError: %s", err)
	}

	// payload contain within the token
	claims := jwt.MapClaims{}

	// Parse the JWT.
	token, err := jwt.ParseWithClaims(t, claims, jwks.Keyfunc)
	if err != nil {
		return nil, fmt.Errorf("could not parse token error='%w'", err)
	}

	// Check if the token is valid.
	if !token.Valid {
		log.Println("token not valid")
		return nil, ErrInvalidToken
	}

	log.Println("The token is valid.")
	return claims, nil
}

func handler(ctx context.Context, event events.APIGatewayWebsocketProxyRequest) (events.APIGatewayCustomAuthorizerResponse, error) {
	token, ok := event.QueryStringParameters["Auth"]

	if !ok || token == "" {
		return events.APIGatewayCustomAuthorizerResponse{}, errors.New("Unauthorized")
	}

	if claims, err := isValid(token); err != nil {
		return events.APIGatewayCustomAuthorizerResponse{}, err
	} else {
		return generatePolicy("user", "Allow", "*", claims), nil
	}
}

func generatePolicy(principalId, effect, resource string, claims jwt.MapClaims) events.APIGatewayCustomAuthorizerResponse {
	authResponse := events.APIGatewayCustomAuthorizerResponse{PrincipalID: principalId}
	if effect != "" && resource != "" {
		authResponse.PolicyDocument = events.APIGatewayCustomAuthorizerPolicy{
			Version: "2012-10-17",
			Statement: []events.IAMPolicyStatement{
				{
					Action:   []string{"execute-api:Invoke"},
					Effect:   effect,
					Resource: []string{resource},
				},
			},
		}
	}

	authResponse.Context = map[string]interface{}{
		"email": claims["email"],
	}

	return authResponse
}
