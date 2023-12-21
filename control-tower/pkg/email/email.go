package email

import (
	"context"
	"errors"

	"github.com/mailgun/mailgun-go/v4"
)

var (
	ErrEmptyEmail = errors.New("empty email not allowed")
)

type EmailSvcOpts struct {
	Domain string `json:"domain"`
	ApiKey string `json:"apiKey"`
}

type Email struct {
	TemplateId string
	Subject    string
	Html       string
	From       string
	Variables  *map[string]any
	To         []string
	Cc         []string
}

type EmailService struct {
	client mailgun.MailgunImpl
}

func NewEmailService(ops *EmailSvcOpts) *EmailService {
	return &EmailService{
		client: *mailgun.NewMailgun(ops.Domain, ops.ApiKey),
	}
}
func NewEmail(templateId, html, subject, from string, variables *map[string]any, to, cc []string) *Email {
	return &Email{
		TemplateId: templateId,
		Html:       html,
		Subject:    subject,
		From:       from,
		Variables:  variables,
		To:         to,
		Cc:         cc,
	}
}

func (s *EmailService) Send(ctx context.Context, email *Email) (err error) {
	if email == nil || (email.Html == "" && email.TemplateId == "") {
		return ErrEmptyEmail
	}
	m := s.client.NewMessage(email.From, email.Subject, email.Html, email.To...)

	if email.Html == "" {
		m.SetTemplate(email.TemplateId)

		for k, v := range *email.Variables {
			err = m.AddTemplateVariable(k, v)

			if err != nil {
				return err
			}
		}
	}
	_, _, err = s.client.Send(ctx, m)
	return err
}
