package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/japb1998/control-tower/internal/service"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

func handler(ctx context.Context, event service.Notification) error {
	c, span := tracer.Start(ctx, "handler-start")
	defer span.End()

	var wg sync.WaitGroup
	validate := validator.New(validator.WithRequiredStructEnabled())

	err := validate.Struct(event)

	if err != nil {

		for _, ve := range err.(validator.ValidationErrors) {
			handlerLogger.Info("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
		}
		return fmt.Errorf("error validation ocurred")
	}

	// get client span
	getClientCtx, getClientSpan := tracer.Start(c, "get-client")
	client, err := clientSvc.GetClientById(getClientCtx, event.CreatedBy, event.ClientId)

	if err != nil {
		getClientSpan.End()
		return err
	}
	if *client.OptIn == false {
		handlerLogger.Info("notification disabled.", slog.String("creator", event.CreatedBy), slog.String("client", event.ClientId))
		getClientSpan.End()
		return nil
	}
	getClientSpan.End()

	_, deliverySpan := tracer.Start(c, "delivery-methods-loop")
	errChan := make(chan error, len(event.DeliveryMethods))
	for _, i := range event.DeliveryMethods {

		switch service.ContactOptions(i) {
		case service.Email:
			if client.Email == "" {
				handlerLogger.Error("Skipping email delivery. reason='email is empty.'")
				errChan <- nil
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, time.Second*30)
				defer cancel()

				opOut := fmt.Sprint("%s/unsubscribe/%s/%s", apiUrl, client.CreatedBy, client.Id)

				if err := sendEmailReminder(ctx, client.FirstName, client.LastName, opOut, []string{client.Email}); err != nil {
					handlerLogger.Info("email failed to send", slog.String("error", err.Error()))
					errChan <- err

				} else {
					errChan <- nil
				}
			}()
		case service.Phone:
			if client.Phone == "" {
				handlerLogger.Info("Skipping whatsapp delivery. reason='phone is empty.'")
				errChan <- nil
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()

				if err := sendWhatsappNotification(client.FirstName, "2", client.Phone); err != nil {
					handlerLogger.Info("failed to send message %s\n", slog.String("error", err.Error()))
					errChan <- err

				} else {
					handlerLogger.Info("SMS was successfully sent", slog.String("to", client.Phone))
					errChan <- nil
				}

			}()

		default:
			handlerLogger.Info("Invalid delivery method method: %d\n", i)
			errChan <- fmt.Errorf("Invalid delivery method")
		}
	}

	status := service.SentStatus
	for i := 0; i < len(event.DeliveryMethods); i++ {
		if err := <-errChan; err != nil {
			status = service.FailedStatus
		}
	}
	if status == service.FailedStatus {
		deliverySpan.SetStatus(codes.Error, err.Error())
	}
	// end delivery span
	deliverySpan.End()

	_, statusSpan := tracer.Start(c, "set-status-span")
	err = notificationSvc.SetNotificationStatus(event.CreatedBy, event.ID, status)

	if err != nil {
		statusSpan.SetStatus(codes.Error, err.Error())
		statusSpan.End()
		return err
	}
	statusSpan.End()
	close(errChan)

	// we are done waiting for the go routine to finish
	wg.Wait()

	// notify active connections - FE.
	wsCtx, wsSpan := tracer.Start(c, "ws-span")
	msg, err := service.NewNotificationUpdateMsg(client.CreatedBy, event.ID).WithAction(service.NotificationUpdatedAction)

	// we do not return an error because an error here does not mean that the delivery failed
	if err != nil {
		handlerLogger.Error("failed to generate new notification msg", slog.String("error", err.Error()))
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		wsSpan.End()
	}
	handlerLogger.Info("WS message='%v'", msg)

	err = connectionSvc.SendWsMessageByEmail(wsCtx, msg)
	// we do not return an error because an error here does not mean that the delivery failed
	if err != nil {
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		handlerLogger.Error("failed to send WS message", slog.String("error", err.Error()))
		wsSpan.End()
	}
	wsSpan.End()

	return nil
}
