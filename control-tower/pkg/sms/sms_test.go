package sms_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/japb1998/control-tower/pkg/sms"
	"github.com/joho/godotenv"
)

func TestMessage(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	fmt.Println(cwd)
	path := path.Join(cwd, "../../test.env")
	err = godotenv.Load(path)
	msgSvc := sms.MusInitMsgSvc(os.Getenv("TWILIO_SERVICE_ID"))
	templateVariables, err := json.Marshal(map[int]string{
		1: "test_client",
	})
	if err != nil {
		t.Fatal(err)
	}
	msg := sms.Msg{
		To:                "+17866565650",
		TemplateId:        os.Getenv("TWILIO_TEMPLATE_ID"),
		TemplateVariables: templateVariables,
	}

	err = msgSvc.SendMessage(&msg)

	if err != nil {
		t.Fatal(err)
	}
}
