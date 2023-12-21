package email_test

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"path"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/japb1998/control-tower/pkg/credentials"
	"github.com/japb1998/control-tower/pkg/email"
	"github.com/joho/godotenv"
)

func TestEmail(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	path := path.Join(cwd, "../../test.env")
	err = godotenv.Load(path)
	if err != nil {
		log.Fatalf("Error loading env vars: %s", err)
	}

	var ops email.EmailSvcOpts

	cm := credentials.NewCredentialsManager(getSess())
	secretId := os.Getenv("MAIL_GUN_SECRET_ID")
	s, err := cm.GetSecret(secretId)

	if err != nil {
		t.Fatalf("failed to get credentials error:%s", err)
	}

	if err := json.Unmarshal([]byte(*s), &ops); err != nil {
		t.Fatalf("failed to get credentials error:%s", err)
	}

	svc := email.NewEmailService(&ops)
	e := email.NewEmail("lashroom", "", "test subject", "no-reply@webdevlife.me", &map[string]any{"op_out_url": "webdevlife.me", "customer_name": "Javier Perez"}, []string{"japb.dev@gmail.com"}, nil)
	err = svc.Send(context.Background(), e)

	if err != nil {
		t.Fatal(err)
	}
}

func getSess() *session.Session {
	var sess *session.Session

	sess = session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Profile:           "personal",
		Config: aws.Config{
			Region: aws.String("us-east-1"),
		},
	}))
	return sess
}
