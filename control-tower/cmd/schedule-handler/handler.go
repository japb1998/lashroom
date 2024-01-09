package main

import (
	"context"
	"fmt"
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
			fmt.Printf("%s validation: %s failed. value='%s', param='%s'\n", ve.Namespace(), ve.Tag(), ve.Value(), ve.Param())
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
		fmt.Printf("client User='%s', clientID='%s' has notifications disabled. Skipping.", event.CreatedBy, event.ClientId)
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
				handlerLogger.Println("Skipping email delivery. reason='email is empty.'")
				errChan <- nil
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				ctx, cancel := context.WithTimeout(ctx, time.Second*30)
				defer cancel()

				opOut := fmt.Sprintf("%s/unsubscribe/%s/%s", apiUrl, client.CreatedBy, client.Id)

				if err := sendEmailReminder(ctx, client.FirstName, client.LastName, opOut, []string{client.Email}); err != nil {
					fmt.Printf("email failed to send error= %s\n", err)
					errChan <- err

				} else {
					errChan <- nil
				}
			}()
		case service.Phone:
			if client.Phone == "" {
				handlerLogger.Println("Skipping whatsapp delivery. reason='phone is empty.'")
				errChan <- nil
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()

				if err := sendWhatsappNotification(client.FirstName, "2", client.Phone); err != nil {
					handlerLogger.Printf("failed to send message %s\n", err)
					errChan <- err

				} else {
					handlerLogger.Printf("SMS was successfully sent to='%s'!\n", client.Phone)
					errChan <- nil
				}

			}()

		default:
			handlerLogger.Printf("Invalid delivery method method: %d\n", i)
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
		handlerLogger.Println(err)
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		handlerLogger.Println(err)
		wsSpan.End()
	}
	handlerLogger.Printf("WS message='%v'", msg)

	err = connectionSvc.SendWsMessageByEmail(wsCtx, msg)
	// we do not return an error because an error here does not mean that the delivery failed
	if err != nil {
		wsSpan.SetAttributes(attribute.KeyValue{
			Key:   attribute.Key("ws-delivery"),
			Value: attribute.BoolValue(false),
		})
		handlerLogger.Println(err)
		wsSpan.End()
	}
	wsSpan.End()

	return nil
}
