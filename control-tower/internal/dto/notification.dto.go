package dto

type PaginatedNotifications struct {
	Limit int            `json:"limit"`
	Page  int            `json:"page"`
	Total int64          `json:"total"`
	Data  []Notification `json:"data"`
}

type NotificationInput struct {
	Date            string            `json:"date" binding:"required,rfc3339"`
	ClientId        string            `json:"clientId" binding:"required"`
	DeliveryMethods []int8            `json:"deliveryMethods" binding:"required,min=1,dive,number,min=0,max=1"`
	Payload         map[string]string `json:"payload,omitempty"`
}
type Notification struct {
	ID              string            `json:"id,omitempty" validate:"required,uuid"`
	Status          string            `json:"status" validate:"required"`
	Date            string            `json:"date" validate:"required"`
	ClientId        string            `json:"clientId" validate:"required,uuid"`
	CreatedBy       string            `json:"createdBy" validate:"required,email"`
	ClientToken     string            `json:"clientToken" validate:"uuid"`
	DeliveryMethods []int8            `json:"deliveryMethods" validate:"required,min=1,dive,min=0,max=1"`
	Payload         map[string]string `json:"payload,omitempty"`
}

// PatchNotification payload for updating a notification.
type PatchNotification struct {
	Date            string `json:"date" binding:"omitempty,rfc3339"`
	ClientId        string `json:"clientId" binding:"omitempty,uuid"`
	Status          string `json:"status"`
	DeliveryMethods []int8 `json:"deliveryMethods,omitempty" binding:"omitempty,dive,number,min=0,max=1"`
}
