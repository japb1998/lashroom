package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/model"
	"github.com/japb1998/control-tower/internal/scheduler"
)

var (
	notificationHandler = os.Getenv("NOTIFICATION_LAMBDA")
	notificationLogger  = log.New(os.Stdout, "[Notification Service]", log.Default().Flags())
)

const (
	thirtyDays  = time.Hour * 24 * 30
	ttlDuration = thirtyDays
)

// used as enums
const (
	Phone ContactOptions = iota
	Email
)
const (
	SentStatus    notificationStatus = "SENT"
	FailedStatus  notificationStatus = "FAILED"
	NotSentStatus notificationStatus = "NOT_SENT"
)

var (
	ErrInvalidMethod        = errors.New("invalid delivery method provided")
	ErrInvalidDate          = errors.New("invalid date provided")
	ErrInvalidStatus        = errors.New("invalid notification status")
	ErrNotificationNotFound = errors.New("notification not found")
)

// ContactOptions are the different type of notifications
type ContactOptions uint8

// notificationStatus marks the status in which the notification is at the moment.
type notificationStatus string

type NotificationRepository interface {
	Delete(createdBy, id string) error
	Create(notification model.NotificationItem) error
	GetNotification(createdBy, name string) (*model.NotificationItem, error)
	GetNotificationsByCreator(createdBy string, ops *database.PaginationOps) (*database.PaginatedNotifications, error)
	UpdateNotification(createdBy, id string, notification database.PatchNotificationItem) (*model.NotificationItem, error)
	SetStatus(partitionKey string, sortKey string, status string) error
}

func (s notificationStatus) String() string {
	return string(s)
}

// PaginationOps. used to determine how items should be returned
type PaginationOps struct {
	Page  int `form:"page" binding:"omitempty,min=0"`
	Limit int `form:"limit" binding:"omitempty,min=1"`
}

type PaginatedNotifications struct {
	Limit int            `json:"limit"`
	Page  int            `json:"page"`
	Total int64          `json:"total"`
	Data  []Notification `json:"data"`
}

type NotificationInput struct {
	Date            string `json:"date" binding:"required,rfc3339"`
	ClientId        string `json:"clientId" binding:"required"`
	DeliveryMethods []int8 `json:"deliveryMethods" binding:"required,min=1,dive,number,min=0,max=1"`
}
type Notification struct {
	ID              string `json:"id,omitempty" validate:"required,uuid"`
	Status          string `json:"status" validate:"required"`
	Date            string `json:"date" validate:"required"`
	ClientId        string `json:"clientId" validate:"required,uuid"`
	CreatedBy       string `json:"createdBy" validate:"required,email"`
	ClientToken     string `json:"clientToken" validate:"uuid"`
	DeliveryMethods []int8 `json:"deliveryMethods" validate:"required,min=1,dive,min=0,max=1"`
}

type PatchNotification struct {
	Date            string `json:"date" binding:"omitempty,rfc3339"`
	ClientId        string `json:"clientId" binding:"omitempty,uuid"`
	Status          string `json:"status"`
	DeliveryMethods []int8 `json:"deliveryMethods,omitempty" binding:"omitempty,dive,number,min=0,max=1"`
}

func NewNotification(createdBy, id, clientToken string, input *NotificationInput) *Notification {
	return &Notification{
		ID:              id,
		Status:          fmt.Sprintf("%s", NotSentStatus),
		DeliveryMethods: input.DeliveryMethods,
		Date:            input.Date,
		ClientId:        input.ClientId,
		CreatedBy:       createdBy,
		ClientToken:     clientToken,
	}
}
func NewNotificationFromItem(item *model.NotificationItem) *Notification {
	return &Notification{
		ID:              item.SortKey,
		Status:          item.Status,
		DeliveryMethods: item.DeliveryMethods,
		Date:            item.Date,
		ClientId:        item.ClientId,
		CreatedBy:       item.PartitionKey,
	}
}

type NotificationService struct {
	store     NotificationRepository
	scheduler scheduler.Scheduler
}

func NewNotificationService(store NotificationRepository, scheduler scheduler.Scheduler) *NotificationService {
	return &NotificationService{
		store,
		scheduler,
	}
}

// DeleteNotification is used to cleanup the notification in the db.
func (s *NotificationService) DeleteNotification(createdBy, id string) error {
	i, err := s.store.GetNotification(createdBy, id)

	if err != nil {
		if errors.Is(err, database.ErrNotificationNotFound) {
			return fmt.Errorf("Error ocurred error='%w'. Notification with ID='%s'", ErrNotificationNotFound, id)
		}
		notificationLogger.Println(err)
		return fmt.Errorf("failed to find notification with ID='%s', err: %w", id, err)
	}

	if i.Status == string(NotSentStatus) {
		err = s.scheduler.DeleteSchedule(id, i.ClientToken)
		if err != nil {
			if !errors.Is(err, scheduler.ErrNotFound) {
				notificationLogger.Printf("error while removing schedule: %s", err)
				return fmt.Errorf("failed to delete schedule")
			}
		}
	}

	err = s.store.Delete(createdBy, id)

	if err != nil {
		notificationLogger.Printf("error while deleting notification from db: %s", err)
		return fmt.Errorf("failed to delete notification")
	}

	return nil
}

// ScheduleNotification schedules a new execution and stores the schedule ID in the db
func (s *NotificationService) ScheduleNotification(createdBy string, notification *NotificationInput) error {
	// we parse the incoming date in UTC
	date, err := time.Parse(time.RFC3339, notification.Date)
	if err != nil {
		return fmt.Errorf("invalid notification date error: %w", err)
	}
	// item uuid is generated here.
	i := model.NewNotificationItem(createdBy, NotSentStatus.String(), notification.ClientId, date, notification.DeliveryMethods)

	newNotification := NewNotification(createdBy, i.SortKey, i.ClientToken, notification)

	payload, err := json.Marshal(newNotification)

	if err != nil {
		notificationLogger.Printf("error while marshalling notification payload: %s", err)
		return fmt.Errorf("error while scheduling notification")
	}

	// create event bridge event
	handler := os.Getenv("NOTIFICATION_LAMBDA")
	role := os.Getenv("SCHEDULER_ROLE")
	sch := scheduler.NewSchedule(i.SortKey, handler, role, scheduler.TimeZoneETD, string(payload), date)
	_, err = s.scheduler.CreateSchedule(sch, i.ClientToken)

	if err != nil {
		notificationLogger.Printf("error while creating schedule: %s", err)
		if errors.Is(err, scheduler.ErrInvalidDate) {
			return ErrInvalidDate
		}
		return fmt.Errorf("error while scheduling notification")
	}
	// store in dynamo .
	err = s.store.Create(*i)

	if err != nil {
		notificationLogger.Printf("error while storing notification: %s", err)
		// clean up the notification
		err = s.scheduler.DeleteSchedule(newNotification.ID, i.ClientToken)
		notificationLogger.Println("failed to cleanup notification err: ", err)

		return fmt.Errorf("error while scheduling notification")
	}

	notificationLogger.Printf("notification created ID: '%s'", newNotification.ID)
	return nil
}

// UpdateNotification function allows you to update all notification fields except status for status use SetNotificationStatus.
func (s *NotificationService) UpdateNotification(createdBy string, name string, ps PatchNotification) (Notification, error) {
	sch, err := s.scheduler.GetSchedule(name)
	var input Notification
	patchItem := database.PatchNotificationItem{}
	if err != nil {

		notificationLogger.Println(err)
		return Notification{}, fmt.Errorf("Unable to get notification")
	}
	err = json.Unmarshal([]byte(sch.Payload), &input)

	if err != nil {
		return Notification{}, fmt.Errorf("error retrieving schedule payload error: %w", err)
	}

	if ps.ClientId != "" {
		input.ClientId = ps.ClientId
	}

	if ps.DeliveryMethods != nil && len(ps.DeliveryMethods) != 0 {
		input.DeliveryMethods = ps.DeliveryMethods
		patchItem.DeliveryMethods = ps.DeliveryMethods
	}
	notificationLogger.Println(ps.Date)
	if ps.Date != "" {
		t, err := time.Parse(time.RFC3339, ps.Date)

		if err != nil {
			return Notification{}, fmt.Errorf("unable to format provided date:  %w", err)
		}

		if t.Before(time.Now()) {
			return Notification{}, fmt.Errorf("date must happen before %s, got %s", time.Now(), ps.Date)
		}
		//  input date field
		input.Date = ps.Date

		// new date for the schedule
		sch.Date = t

		//database item
		patchItem.Date = t
	}

	// new schedule payload
	payload, err := json.Marshal(input)

	if err != nil {
		return Notification{}, fmt.Errorf("error setting new payload err: %w", err)
	}
	sch.Payload = string(payload)
	_, err = s.scheduler.UpdateSchedule(sch)

	if err != nil {
		return Notification{}, nil
	}

	_, err = s.store.UpdateNotification(createdBy, name, patchItem)

	if err != nil {
		notificationLogger.Println(err)
		// TODO: if error when updating the db -  rollback
		return Notification{}, fmt.Errorf("DB failed to updated")
	}
	return input, nil
}

// SetNotificationStatus changes the notification status
func (s *NotificationService) SetNotificationStatus(createdBy string, id string, status notificationStatus) error {
	err := s.store.SetStatus(createdBy, id, string(status))
	if err != nil {
		notificationLogger.Println(err)
		return fmt.Errorf("error when setting status")
	}
	return nil
}

// GetNotification gets notification by dynamo Key
func (s *NotificationService) GetNotification(createdBy, name string) (*Notification, error) {
	n, err := s.store.GetNotification(createdBy, name)

	if err != nil {
		if errors.Is(err, database.ErrNotificationNotFound) {
			notificationLogger.Printf("Notification ID='%s' NOT FOUND", name)
			return nil, fmt.Errorf("error occurred error: %w", ErrNotificationNotFound)
		}
		notificationLogger.Printf("Error: %s", err)
		return nil, fmt.Errorf("error getting notification")
	}

	notification := Notification{
		ID:              n.SortKey,
		Status:          n.Status,
		Date:            n.Date,
		ClientId:        n.ClientId,
		CreatedBy:       n.PartitionKey,
		DeliveryMethods: n.DeliveryMethods,
		ClientToken:     n.ClientToken,
	}
	return &notification, nil
}

// GetNotificationsByCreator gets notification by creator
func (s *NotificationService) GetNotificationsByCreator(createdBy string, ops *PaginationOps) (*PaginatedNotifications, error) {
	dbOps := database.PaginationOps{
		Skip:  ops.Limit * ops.Page,
		Limit: ops.Limit,
	}
	notificationLogger.Printf("skip %d", ops.Limit*ops.Page)
	paginatedItems, err := s.store.GetNotificationsByCreator(createdBy, &dbOps)

	if err != nil {
		notificationLogger.Printf("Error: %s", err)
		return nil, fmt.Errorf("error while getting notifications creator: %s, error: %w", createdBy, err)
	}

	notifications := make([]Notification, 0, len(paginatedItems.Data))

	for i := 0; i < len(paginatedItems.Data); i++ {
		n := (paginatedItems.Data)[i]
		notification := Notification{
			ID:              n.SortKey,
			Status:          n.Status,
			Date:            n.Date,
			ClientId:        n.ClientId,
			CreatedBy:       n.PartitionKey,
			DeliveryMethods: n.DeliveryMethods,
			ClientToken:     n.ClientToken,
		}
		notifications = append(notifications, notification)
	}

	return &PaginatedNotifications{
		Total: paginatedItems.Total,
		Page:  ops.Page,
		Data:  notifications,
	}, nil
}
