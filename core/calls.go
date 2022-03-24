package core

import "fmt"

type Call struct {
	GUID       string `json:"guid"`
	Name       string `json:"name"`
	URL        string `json:"url"`
	AuthHeader string `json:"auth_header"`

	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`

	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}

func (entity *Call) Type() string {
	return "call"
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
