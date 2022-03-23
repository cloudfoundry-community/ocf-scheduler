package core

type Schedule struct {
	GUID           string `json:"guid"`
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	RefGUID string `json:"-"`
	RefType string `json:"-"`
}

type ScheduleService interface {
	Persist(*Schedule) (*Schedule, error)
	ByCall(*Call) []*Schedule
	ByJob(*Job) []*Schedule
	Get(string) (*Schedule, error)
	Delete(*Schedule) error
}
