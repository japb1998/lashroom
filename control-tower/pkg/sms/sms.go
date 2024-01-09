package sms

import (
	"fmt"

	"github.com/go-playground/validator/v10"
	twilio "github.com/twilio/twilio-go"
	openapi "github.com/twilio/twilio-go/rest/api/v2010"
)

type MsgSvc struct {
	Client             *twilio.RestClient
	MessagingServiceId string `validate:"required"` //twilio service id
}

// Msg. TemplateVariables is the json containing the variables to be replaces in the template
type Msg struct {
	TemplateId        string // whatsapp
	TemplateVariables []byte // json
	To                string `validate:"e164"`
}

func (svc *MsgSvc) SendMessage(msg *Msg) error {
	params := &openapi.CreateMessageParams{}
	params.SetTo(fmt.Sprintf("whatsapp:%s", msg.To))
	params.SetFrom(svc.MessagingServiceId)
	params.SetContentSid(msg.TemplateId)

	if msg.TemplateVariables != nil {
		params.SetContentVariables(string(msg.TemplateVariables))
	}

	_, err := svc.Client.Api.CreateMessage(params)
	if err != nil {
		fmt.Println(err.Error())
		return fmt.Errorf("failed to send Whatsapp message! error: %w", err)
	}
	return nil
}

// MustInitSvc returns a MsgSvc or panics if error.
func MusInitMsgSvc(serviceId string) *MsgSvc {
	svc := &MsgSvc{
		Client:             twilio.NewRestClient(),
		MessagingServiceId: serviceId,
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
