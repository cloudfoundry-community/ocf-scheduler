package presenters

import "github.com/starkandwayne/scheduler-for-ocf/core"

func AsJobSchedule(schedule *core.Schedule) *jobSchedule {
	output := &jobSchedule{
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

type jobSchedule struct {
	GUID           string `json:"guid"`
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	JobGUID string `json:"job_guid"`
}
