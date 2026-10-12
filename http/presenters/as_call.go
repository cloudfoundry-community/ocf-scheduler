package presenters

import (
	"time"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// AsCall presents a call without its auth header. The header is stored and
// sent when the call runs, but no API response ever returns it.
func AsCall(call *core.Call) *Call {
	return &Call{
		GUID:      call.GUID,
		Name:      call.Name,
		URL:       call.URL,
		CreatedAt: call.CreatedAt,
		UpdatedAt: call.UpdatedAt,
		AppGUID:   call.AppGUID,
		SpaceGUID: call.SpaceGUID,
	}
}

func AsCallCollection(calls []*core.Call) []*Call {
	output := make([]*Call, 0)

	for _, call := range calls {
		output = append(output, AsCall(call))
	}

	return output
}

type Call struct {
	GUID string `json:"guid"`
	Name string `json:"name"`
	URL  string `json:"url"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	AppGUID   string `json:"app_guid"`
	SpaceGUID string `json:"space_guid"`
}
