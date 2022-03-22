package core

type Job struct {
	GUID       string `json:"guid"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	DiskInMb   int    `json:"disk_in_mb"`
	MemoryInMb int    `json:"memory_in_mb"`
	State      string `json:"state"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}

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

type JobService interface {
	Get(string) (*Job, error)
	Delete(*Job) error
	Named(string) (*Job, error)
	Persist(*Job) (*Job, error)
	InSpace(string) []*Job
}

type EnvironmentInfoService interface {
	SpaceGUID() string
}

type ScheduleService interface {
	Persist(*Schedule) (*Schedule, error)
	ByJob(*Job) []*Schedule
}

type Services struct {
	Environment EnvironmentInfoService
	Jobs        JobService
	Schedules   ScheduleService
}
