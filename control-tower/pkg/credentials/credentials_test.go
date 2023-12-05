package credentials_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/japb1998/control-tower/pkg/credentials"
	"github.com/joho/godotenv"
)

func TestRetrieval(t *testing.T) {
	var sess *session.Session
	switch os.Getenv("STAGE") {
	case "LOCAL":
		err := godotenv.Load("../../test.env")
		if err != nil {
			t.Fatalf("Error loading env vars: %s", err)
		}

		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
			Profile:           "personal",
			Config: aws.Config{
				Region: aws.String("us-east-1"),
			},
		}))
	default:
		sess = session.Must(session.NewSessionWithOptions(session.Options{
			SharedConfigState: session.SharedConfigEnable,
		}))
	}
	cm := credentials.NewCredentialsManager(sess)
	arn := os.Getenv("MAIL_GUN_SECRET_ID")
	b, err := cm.GetSecret(arn)

	if err != nil {
		t.Fatalf("failed to get secret error='%s'", err)
	}
	var secret map[string]any

	if err := json.Unmarshal([]byte(*b), &secret); err != nil {
		t.Fatalf("failed to unmarshall secret error='%s' payload='%s'", err, *b)
	}

	fmt.Println(secret)
}
