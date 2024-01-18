package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/japb1998/control-tower/internal/database"
	"github.com/japb1998/control-tower/internal/dto"
	"github.com/japb1998/control-tower/internal/model"
	"github.com/japb1998/control-tower/internal/scheduler"
)

var (
	notificationHandler = os.Getenv("NOTIFICATION_LAMBDA")
)

const (
	thirtyDays  = time.Hour * 24 * 30
	ttlDuration = thirtyDays
)

// used as enums
const (
	Whatsapp ContactOptions = iota
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

func NewNotification(createdBy, id, clientToken string, input *dto.NotificationInput) *dto.Notification {
	return &dto.Notification{
		ID:              id,
		Status:          fmt.Sprintf("%s", NotSentStatus),
		DeliveryMethods: input.DeliveryMethods,
		Date:            input.Date,
		ClientId:        input.ClientId,
		CreatedBy:       createdBy,
		ClientToken:     clientToken,
		Payload:         input.Payload,
	}
}
func NewNotificationFromItem(item *model.NotificationItem) *dto.Notification {
	return &dto.Notification{
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
		notificationLogger.Error(err.Error())
		return fmt.Errorf("failed to find notification with ID='%s', err: %w", id, err)
	}

	if i.Status == string(NotSentStatus) {
		err = s.scheduler.DeleteSchedule(id, i.ClientToken)
		if err != nil {
			if !errors.Is(err, scheduler.ErrNotFound) {
				notificationLogger.Error("error while removing schedule", slog.String("error", err.Error()))
				return fmt.Errorf("failed to delete schedule")
			}
		}
	}

	err = s.store.Delete(createdBy, id)

	if err != nil {
		notificationLogger.Error("error while deleting notification from db", slog.String("error", err.Error()))
		return fmt.Errorf("failed to delete notification")
	}

	return nil
}

// ScheduleNotification schedules a new execution and stores the schedule ID in the db
func (s *NotificationService) ScheduleNotification(createdBy string, notification *dto.NotificationInput) (id string, err error) {
	// we parse the incoming date in UTC
	date, err := time.Parse(time.RFC3339, notification.Date)
	if err != nil {
		return "", fmt.Errorf("invalid notification date error: %w", err)
	}
	// item uuid is generated here.
	i := model.NewNotificationItem(createdBy, NotSentStatus.String(), notification.ClientId, date, notification.DeliveryMethods)
	// set id to be returned here.
	id = i.SortKey

	newNotification := NewNotification(createdBy, i.SortKey, i.ClientToken, notification)

	payload, err := json.Marshal(newNotification)

	if err != nil {
		notificationLogger.Error("error while marshalling notification payload", slog.String("error", err.Error()))
		return "", fmt.Errorf("error while scheduling notification")
	}

	// create event bridge event
	handler := os.Getenv("NOTIFICATION_LAMBDA")
	role := os.Getenv("SCHEDULER_ROLE")
	sch := scheduler.NewSchedule(i.SortKey, handler, role, scheduler.TimeZoneETD, string(payload), date)
	_, err = s.scheduler.CreateSchedule(sch, i.ClientToken)

	if err != nil {
		notificationLogger.Error("error while creating schedule", slog.String("error", err.Error()))
		if errors.Is(err, scheduler.ErrInvalidDate) {
			return "", ErrInvalidDate
		}
		return "", fmt.Errorf("error while scheduling notification")
	}
	// store in dynamo .
	err = s.store.Create(*i)

	if err != nil {
		notificationLogger.Error("error while storing notification", slog.String("error", err.Error()))
		// clean up the notification
		err = s.scheduler.DeleteSchedule(newNotification.ID, i.ClientToken)
		notificationLogger.Error("failed to cleanup notification", slog.String("error", err.Error()))

		return "", fmt.Errorf("error while scheduling notification")
	}

	notificationLogger.Info("notification created", slog.String("notificationId", newNotification.ID))

	return id, nil
}

// UpdateNotification function allows you to update all notification fields except status for status use SetNotificationStatus.
func (s *NotificationService) UpdateNotification(createdBy string, name string, ps dto.PatchNotification) (dto.Notification, error) {
	sch, err := s.scheduler.GetSchedule(name)
	var input dto.Notification
	patchItem := database.PatchNotificationItem{}
	if err != nil {

		notificationLogger.Error(err.Error())
		return dto.Notification{}, fmt.Errorf("Unable to get notification")
	}
	err = json.Unmarshal([]byte(sch.Payload), &input)

	if err != nil {
		return dto.Notification{}, fmt.Errorf("error retrieving schedule payload error: %w", err)
	}

	if ps.ClientId != "" {
		input.ClientId = ps.ClientId
	}

	if ps.DeliveryMethods != nil && len(ps.DeliveryMethods) != 0 {
		input.DeliveryMethods = ps.DeliveryMethods
		patchItem.DeliveryMethods = ps.DeliveryMethods
	}
	if ps.Date != "" {
		t, err := time.Parse(time.RFC3339, ps.Date)

		if err != nil {
			return dto.Notification{}, fmt.Errorf("unable to format provided date:  %w", err)
		}

		if t.Before(time.Now()) {
			return dto.Notification{}, fmt.Errorf("date must happen before %s, got %s", time.Now(), ps.Date)
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
		return dto.Notification{}, fmt.Errorf("error setting new payload err: %w", err)
	}
	sch.Payload = string(payload)
	_, err = s.scheduler.UpdateSchedule(sch)

	if err != nil {
		return dto.Notification{}, nil
	}

	_, err = s.store.UpdateNotification(createdBy, name, patchItem)

	if err != nil {
		notificationLogger.Error(err.Error())
		// TODO: if error when updating the db -  rollback
		return dto.Notification{}, fmt.Errorf("DB failed to updated")
	}
	return input, nil
}

// SetNotificationStatus changes the notification status
func (s *NotificationService) SetNotificationStatus(createdBy string, id string, status notificationStatus) error {
	err := s.store.SetStatus(createdBy, id, string(status))
	if err != nil {
		notificationLogger.Error(err.Error())
		return fmt.Errorf("error when setting status")
	}
	return nil
}

// GetNotification gets notification by dynamo Key
func (s *NotificationService) GetNotification(createdBy, name string) (*dto.Notification, error) {
	n, err := s.store.GetNotification(createdBy, name)

	if err != nil {
		if errors.Is(err, database.ErrNotificationNotFound) {
			notificationLogger.Error("Notification NOT FOUND", name)
			return nil, fmt.Errorf("error occurred error: %w", ErrNotificationNotFound)
		}
		notificationLogger.Error("Error getting notification", slog.String("error", err.Error()))
		return nil, fmt.Errorf("error getting notification")
	}

	notification := dto.Notification{
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
func (s *NotificationService) GetNotificationsByCreator(createdBy string, ops *dto.PaginationOps) (*dto.PaginatedNotifications, error) {
	dbOps := database.PaginationOps{
		Skip:  *ops.Limit * ops.Page,
		Limit: *ops.Limit,
	}
	notificationLogger.Info("skip", slog.Int("skip", *ops.Limit*ops.Page))
	paginatedItems, err := s.store.GetNotificationsByCreator(createdBy, &dbOps)

	if err != nil {
		notificationLogger.Error("Error", slog.String("error", err.Error()))
		return nil, fmt.Errorf("error while getting notifications creator: %s, error: %w", createdBy, err)
	}

	notifications := make([]dto.Notification, 0, len(paginatedItems.Data))

	for i := 0; i < len(paginatedItems.Data); i++ {
		n := (paginatedItems.Data)[i]
		notification := dto.Notification{
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

	return &dto.PaginatedNotifications{
		Total: paginatedItems.Total,
		Page:  ops.Page,
		Data:  notifications,
	}, nil
}
