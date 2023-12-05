package model

import (
	"time"

	"github.com/google/uuid"
)

// NotificationItem is the entity that will be stored in dynamo
type NotificationItem struct {
	PartitionKey    string `json:"primaryKey"` //createdBy
	SortKey         string `json:"sortKey"`    // ID this will be the schedule name
	Status          string `json:"status"`
	ClientId        string `json:"clientId"`
	Date            string `json:"date"`
	DeliveryMethods []int8 `json:"deliveryMethods"`
	ClientToken     string `json:"clientToken"`
	TTL             int64  `json:"TTL"` //  time to live
}

func NewNotificationItem(pk, status, clientId string, date time.Time, deliveryMethods []int8) *NotificationItem {
	return &NotificationItem{
		PartitionKey:    pk,
		SortKey:         uuid.New().String(),
		ClientId:        clientId,
		Date:            date.Format(time.RFC3339),
		DeliveryMethods: deliveryMethods,
		Status:          status,
		ClientToken:     uuid.New().String(),
		TTL:             date.Add(time.Hour * 24).Unix(), // one day after the notification is sent

	}
}
