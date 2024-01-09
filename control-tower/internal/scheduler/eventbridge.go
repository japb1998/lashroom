// scheduler package is in charge of scheduling notifications using aws event bridge
package scheduler

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	awsScheduler "github.com/aws/aws-sdk-go/service/scheduler"
)

var (
	TimeZoneETD = "America/New_York"
	TimeZoneECT = "America/Los_Angeles"
)

var (
	ErrInvalidDate = errors.New("Invalid Date was provided")
	ErrInvalidTZ   = errors.New("Invalid time zone provided")
	ErrNotFound    = errors.New("Schedule Not Found")
)

type schedule struct {
	Name     string
	TimeZone string
	Payload  string
	Role     string
	Target   string
	Date     time.Time
}

type scheduler struct {
	ebScheduler *awsScheduler.Scheduler
}

func NewScheduler(sess *session.Session) *scheduler {

	s := awsScheduler.New(sess)

	return &scheduler{
		ebScheduler: s,
	}
}

func NewSchedule(name, targetID, role, tz, payload string, date time.Time) *schedule {
	return &schedule{
		Name:     name,
		TimeZone: tz,
		Payload:  payload,
		Role:     role,
		Date:     date, // date in UTC
		Target:   targetID,
	}
}

// creates a schedule using aws eventbridge and returns the schedule name. important: schedule name must be unique.
func (s *scheduler) CreateSchedule(sch *schedule, token string) (name string, err error) {

	var expression string
	var loc *time.Location
	// check current time
	switch sch.TimeZone {
	case TimeZoneETD:
		loc, err = time.LoadLocation(TimeZoneETD)

		if err != nil {
			return "", err
		}

	case TimeZoneECT:
		loc, err = time.LoadLocation(TimeZoneETD)

		if err != nil {
			return "", err
		}

	default:
		return "", ErrInvalidTZ
	}
	if sch.Date.Before(time.Now().Add(time.Second * 30)) {
		return "", fmt.Errorf("date must be after at least 30s greater than the current time, Error: %w", ErrInvalidDate)
	}
	expression = fmt.Sprintf("at(%s)", sch.Date.In(loc).Format("2006-01-02T15:04:05"))
	target := &awsScheduler.Target{
		Arn:     &sch.Target,
		RoleArn: &sch.Role,
		Input:   &sch.Payload,
		RetryPolicy: &awsScheduler.RetryPolicy{
			MaximumRetryAttempts: aws.Int64(0),
		},
	}

	input := &awsScheduler.CreateScheduleInput{
		Name:                       &sch.Name,
		ScheduleExpression:         &expression,
		ActionAfterCompletion:      aws.String("DELETE"),
		Target:                     target,
		ScheduleExpressionTimezone: &sch.TimeZone,
		FlexibleTimeWindow: &awsScheduler.FlexibleTimeWindow{
			Mode: aws.String("OFF"),
		},
		ClientToken: &token,
	}
	_, err = s.ebScheduler.CreateSchedule(input)

	if err != nil {
		return "", err
	}
	return sch.Name, nil
}

func (s *scheduler) DeleteSchedule(name, token string) error {
	input := &awsScheduler.DeleteScheduleInput{
		Name:        &name,
		ClientToken: &token,
	}
	_, err := s.ebScheduler.DeleteSchedule(input)
	var notFound *awsScheduler.ResourceNotFoundException
	if errors.As(err, &notFound) {
		return ErrNotFound
	}
	return err
}

func (s *scheduler) GetSchedule(name string) (*schedule, error) {
	input := &awsScheduler.GetScheduleInput{
		Name: aws.String(name),
	}

	output, err := s.ebScheduler.GetSchedule(input)

	if err != nil {
		var notFound *awsScheduler.ResourceNotFoundException
		if errors.As(err, &notFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	expression, err := time.Parse("2006-01-02T15:04:05", strings.TrimSuffix(strings.TrimPrefix(*output.ScheduleExpression, "at("), ")"))

	if err != nil {
		return nil, fmt.Errorf("error while parsing output expression error: %w", err)
	}
	sch := NewSchedule(*output.Name, *output.Target.Arn, *output.Target.RoleArn, *output.ScheduleExpressionTimezone, *output.Target.Input, expression)

	return sch, nil
}

func (s *scheduler) UpdateSchedule(sch *schedule) (name string, err error) {
	expression := fmt.Sprintf("at(%s)", sch.Date.Format("2006-01-02T15:04:05"))
	input := &awsScheduler.UpdateScheduleInput{
		Name: &sch.Name,
		Target: &awsScheduler.Target{
			Arn:     &sch.Target,
			RoleArn: &sch.Role,
			Input:   &sch.Payload,
		},
		ScheduleExpression:         &expression,
		ActionAfterCompletion:      aws.String("DELETE"),
		ScheduleExpressionTimezone: &sch.TimeZone,
		FlexibleTimeWindow: &awsScheduler.FlexibleTimeWindow{
			Mode: aws.String("OFF"),
		},
	}

	_, err = s.ebScheduler.UpdateSchedule(input)

	if err != nil {
		return "", err
	}

	return sch.Name, nil

}
