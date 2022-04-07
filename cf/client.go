package cf

import (
	"net/url"

	cf "github.com/cloudfoundry-community/go-cfclient"
)

// Client is an interface that describes the parts of the CF go API that we
// care about
type Client interface {
	AppByGuid(string) (cf.App, error)
	CreateTask(cf.TaskRequest) (cf.Task, error)
	GetTaskByGuid(string) (cf.Task, error)
	ListUsersByQuery(url.Values) (cf.Users, error)
	ListV3RolesByQuery(url.Values) ([]cf.V3Role, error)
}
