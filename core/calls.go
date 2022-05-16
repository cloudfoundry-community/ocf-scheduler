package core

import (
	"fmt"
	"time"
)

type Call struct {
	GUID       string `json:"guid"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	AuthHeader string `json:"auth_header"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}

func (entity *Call) RefType() string {
	return "call"
}

func (entity *Call) RefGUID() string {
	return entity.GUID
}

func (entity *Call) ToJob() (*Job, error) {
	return nil, fmt.Errorf("cannot convert to Job")
}

func (entity *Call) ToCall() (*Call, error) {
	return entity, nil
}

type CallService interface {
	Get(string) (*Call, error)
	Delete(*Call) error
	Named(string) (*Call, error)
	Persist(*Call) (*Call, error)
	InSpace(string) []*Call
}
