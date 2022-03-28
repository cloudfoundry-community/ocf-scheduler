package core

import (
	"fmt"
	"time"
)

type Job struct {
	GUID       string `json:"guid"`
	Name       string `json:"name"`
	Command    string `json:"command"`
	DiskInMb   int    `json:"disk_in_mb"`
	MemoryInMb int    `json:"memory_in_mb"`
	State      string `json:"state"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}

func (entity *Job) Type() string {
	return "job"
}

func (entity *Job) ToJob() (*Job, error) {
	return entity, nil
}

func (entity *Job) ToCall() (*Call, error) {
	return nil, fmt.Errorf("cannot convert to Call")
}

type JobService interface {
	Get(string) (*Job, error)
	Delete(*Job) error
	Named(string) (*Job, error)
	Persist(*Job) (*Job, error)
	InSpace(string) []*Job
	Success(*Job) (*Job, error)
	Fail(*Job) (*Job, error)
}
