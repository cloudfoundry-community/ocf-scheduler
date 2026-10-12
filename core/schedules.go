package core

import "time"

type Schedule struct {
	GUID           string `json:"guid"`
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	RefGUID string `json:"-"`
	RefType string `json:"-"`

	// PausedBy is PausedByUnbind when the service broker disabled the
	// schedule, and empty otherwise. Callers cannot set it.
	PausedBy string `json:"-"`
}

type ScheduleService interface {
	Persist(*Schedule) (*Schedule, error)
	ByCall(*Call) []*Schedule
	ByJob(*Job) []*Schedule
	Get(string) (*Schedule, error)
	Delete(*Schedule) error
	Enabled() []*Schedule

	// ByApp returns every schedule on the app's jobs and calls.
	ByApp(appGUID string) ([]*Schedule, error)

	// Pause disables every enabled schedule on the app's jobs and calls,
	// marks each one with reason, and returns the schedules it changed.
	Pause(appGUID string, reason string) ([]*Schedule, error)

	// Resume enables each schedule named in guids that is on the app's jobs
	// and calls and still carries reason, clears the mark, and returns the
	// schedules it changed.
	Resume(appGUID string, reason string, guids []string) ([]*Schedule, error)
}
