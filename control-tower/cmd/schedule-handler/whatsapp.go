package main

import (
	"encoding/json"
	"os"

	"github.com/japb1998/control-tower/pkg/sms"
)

func sendWhatsappNotification(firstName, weeks, phone string) error {
	templateVariables, err := json.Marshal(map[int]string{
		1: firstName,
		2: "2",
	})
	if err != nil {

		return err
	}

	msg := sms.Msg{
		To:                phone,
		TemplateVariables: templateVariables,
		TemplateId:        os.Getenv("TWILIO_TEMPLATE_ID"),
	}

	return msgSvc.SendMessage(&msg)
}
