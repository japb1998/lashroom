package dbmodule

import "github.com/aws/aws-sdk-go/aws/session"

var Session = session.Must(session.NewSessionWithOptions(session.Options{
	SharedConfigState: session.SharedConfigEnable,
}))
