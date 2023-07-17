package notification

import "log"

type NotificationRepository interface {
	DeleteNotification(createdBy, id string) error
}

type NotificationService struct {
	store NotificationRepository
}

func NewNotificationService(store NotificationRepository) *NotificationService {
	return &NotificationService{
		store,
	}
}

func (s NotificationService) DeleteNotification(createdBy, id string) error {
	err := s.store.DeleteNotification(createdBy, id)

	if err != nil {
		log.Println(err.Error())
		return err
	}

	return nil
}
