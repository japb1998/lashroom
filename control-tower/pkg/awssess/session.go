package awssess

import (
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
)

var sess *session.Session

func MustGetSession() *session.Session {

	if sess != nil {
		return sess
	}

	switch os.Getenv("STAGE") {
	case "local":

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
	return sess
}
