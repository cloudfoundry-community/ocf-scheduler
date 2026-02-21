package cf

import (
	"context"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

// CFClient defines the subset of CF API operations used by this application.
type CFClient interface {
	GetApp(ctx context.Context, guid string) (*resource.App, error)
	CreateTask(ctx context.Context, appGUID string, r *resource.TaskCreate) (*resource.Task, error)
	GetTask(ctx context.Context, guid string) (*resource.Task, error)
	ListUsers(ctx context.Context, opts *cfclient.UserListOptions) ([]*resource.User, error)
	ListRoles(ctx context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error)
}
