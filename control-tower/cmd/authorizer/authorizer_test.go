package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-lambda-go/events"
	"github.com/joho/godotenv"
)

func TestAuthorizer(t *testing.T) {
	setupEnv(t)
	token := os.Getenv("TOKEN")

	res, err := handler(context.Background(), events.APIGatewayWebsocketProxyRequest{
		Headers: map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", token),
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	d, _ := json.MarshalIndent(res, "", " ")
	fmt.Println(string(d))
}

func setupEnv(t *testing.T) {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := path.Join(cwd, "../../test.env")
	err = godotenv.Load(path)
	if err != nil {
		t.Fatalf("Error loading env vars: %s", err)
	}
	// since this are globally set and we load environment vars after import then we need to assign them to the new values.
	regionID = os.Getenv("AWS_REGION")
	accountId = os.Getenv("AWS_ACCOUNT_ID")
	apiId = os.Getenv("API_ID")
}
