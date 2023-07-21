package presenters

import (
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func AsJobSchedule(schedule *core.Schedule) *JobSchedule {
	output := &JobSchedule{
		GUID:           schedule.GUID,
		Enabled:        schedule.Enabled,
		Expression:     schedule.Expression,
		ExpressionType: schedule.ExpressionType,
		CreatedAt:      schedule.CreatedAt,
		UpdatedAt:      schedule.UpdatedAt,
		JobGUID:        schedule.RefGUID,
	}

	return output
}

func AsJobScheduleCollection(schedules []*core.Schedule) []*JobSchedule {
	output := make([]*JobSchedule, 0)

	for _, schedule := range schedules {
		output = append(output, AsJobSchedule(schedule))
	}

	return output
}

type JobSchedule struct {
	GUID           string `json:"guid"`
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	JobGUID string `json:"job_guid"`
}
