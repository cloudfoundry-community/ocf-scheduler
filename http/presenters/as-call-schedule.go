package presenters

import (
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

func AsCallSchedule(schedule *core.Schedule) *CallSchedule {
	output := &CallSchedule{
		GUID:           schedule.GUID,
		Enabled:        schedule.Enabled,
		Expression:     schedule.Expression,
		ExpressionType: schedule.ExpressionType,
		CreatedAt:      schedule.CreatedAt,
		UpdatedAt:      schedule.UpdatedAt,
		CallGUID:       schedule.RefGUID,
	}

	return output
}

func AsCallScheduleCollection(schedules []*core.Schedule) []*CallSchedule {
	output := make([]*CallSchedule, 0)

	for _, schedule := range schedules {
		output = append(output, AsCallSchedule(schedule))
	}

	return output
}

type CallSchedule struct {
	GUID           string `json:"guid"`
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	CallGUID string `json:"call_guid"`
}
