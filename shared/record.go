package shared

import (
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/aws/aws-sdk-go/service/dynamodb/dynamodbattribute"
	"github.com/google/uuid"
)

type ContactOptions uint8
type Record struct {
	ID               *string `json:"id,omitempty"`
	Email            *string `json:"email,omitempty"`
	PhoneNumber      *string `json:"phone,omitempty"`
	Status           *string `json:"status,omitempty"`
	DeliveryMethods  []int8  `json:"deliveryMethods"`
	Date             string  `json:"date"`
	ClientName       string  `json:"clientName"`
	NextNotification *int8   `json:"nextNotification,omitempty"`
	CreatedBy        string  `json:"createdBy"`
}

type PatchRecord struct {
	ID              string  `json:"id"`
	DeliveryMethods []int8  `json:"deliveryMethods,omitempty"`
	Date            *string `json:"date,omitempty"`
}
type NewSchedule struct {
	PrimaryKey      string  `json:"primaryKey"` //UUID
	SortKey         string  `json:"sortKey"`
	Status          string  `json:"status"`
	Email           *string `json:"email,omitempty"`
	PhoneNumber     *string `json:"phone,omitempty"`
	ClientName      string  `json:"clientName"`
	Date            string  `json:"date"`
	DeliveryMethods []int8  `json:"deliveryMethods"`
	TTL             int64   `json:"TTL"`
}

const (
	PHONE ContactOptions = iota
	EMAIL
)
const (
	SENT     = "SENT"
	FAILED   = "FAILED"
	NOT_SENT = "NOT_SENT"
)

func (nr *Record) ToNewSchedule() (*NewSchedule, error) {
	var interval uint8

	if nr.NextNotification != nil {
		interval = uint8(*nr.NextNotification)
	} else {
		interval = 15
	}

	newSchedule := time.Now().UTC().AddDate(0, 0, int(interval))
	sevenDayFromNow := newSchedule.Format(time.RFC3339)
	schedule := NewSchedule{
		Date:        sevenDayFromNow,
		Status:      NOT_SENT,
		ClientName:  nr.ClientName,
		TTL:         newSchedule.AddDate(0, 0, 1).Unix(),
		PhoneNumber: nr.PhoneNumber,
		Email:       nr.Email,
	}
	schedule.SortKey = uuid.New().String()
	schedule.PrimaryKey = nr.CreatedBy
	schedule.DeliveryMethods = make([]int8, 0)
	if nr.DeliveryMethods == nil || len(nr.DeliveryMethods) == 0 {
		return nil, errors.New("delivery Method can't be empty")
	} else {
		for _, dm := range nr.DeliveryMethods {
			switch int8(dm) {
			case int8(PHONE), int8(EMAIL):
				if (dm == int8(PHONE) && nr.PhoneNumber == nil) || (dm == int8(EMAIL) && nr.Email == nil) {
					return nil, errors.New("invalid Delivery method Configuration")
				}
				schedule.DeliveryMethods = append(schedule.DeliveryMethods, int8(dm))
			default:
				return nil, errors.New("invalid delivery method provided")
			}
		}
	}

	return &schedule, nil
}
func (nr *Record) SetCreatedBy(createdBy string) {
	nr.CreatedBy = createdBy
}

func (ns *NewSchedule) ToDynamoAttr() (map[string]*dynamodb.AttributeValue, error) {

	if output, err := dynamodbattribute.MarshalMap(*ns); err != nil {
		log.Println(err)
		return nil, errors.New("failed to marshall new schedule")
	} else {
		return output, nil
	}

}

func (ns *NewSchedule) ToRecord() Record {
	record := Record{
		ID:              &ns.SortKey,
		CreatedBy:       ns.PrimaryKey,
		ClientName:      ns.ClientName,
		Status:          &ns.Status,
		Date:            ns.Date,
		Email:           ns.Email,
		PhoneNumber:     ns.PhoneNumber,
		DeliveryMethods: ns.DeliveryMethods,
	}

	return record
}

func Contains[T comparable](s []T, e T) bool {
	for _, v := range s {
		if v == e {
			return true
		}
	}
	return false
}
