package scheduler

type Scheduler interface {
	CreateSchedule(*schedule, string) (string, error)
	DeleteSchedule(name, token string) error
	GetSchedule(name string) (*schedule, error)
	UpdateSchedule(sch *schedule) (string, error)
}
