package cf

import (
	cf "github.com/cloudfoundry-community/go-cfclient"
)

// Client is an interface that describes the parts of the CF go API that we
// care about
type Client interface {
	AppByGuid(string) (cf.App, error)
	CreateTask(cf.TaskRequest) (cf.Task, error)
	GetTaskByGuid(string) (cf.Task, error)
}
