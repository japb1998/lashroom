package sms

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type MsgSvc struct {
	Client       *twilio.RestClient
	SenderNumber string `validate:"e164"` //sender phone number
}

type Msg struct {
	Body string `validate:"min=2"`
	To   string `validate:"e164"`
}

func (svc *MsgSvc) sendMessage(msg *Msg) error {
	params := &openapi.CreateMessageParams{}
	params.SetTo(fmt.Sprintf("whatsapp:%s", msg.To))
	params.SetFrom(fmt.Sprintf("whatsapp:%s", svc.SenderNumber))
	params.SetBody(msg.Body)

	_, err := svc.Client.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("failed to send Whatsapp message! error: %w", err)
	}
	return nil
}

// MustInitSvc returns a MsgSvc or panics if error.
func MusInitMsgSvc(twilioN string) *MsgSvc {
	svc := &MsgSvc{
		Client:       twilio.NewRestClient(),
		SenderNumber: twilioN,
	}
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(svc)

	if err != nil {

		for _, ve := range err.(validator.ValidationErrors) {
			fmt.Printf("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
		}
		panic("failed to setup messaging service")
	}

	return svc
}
