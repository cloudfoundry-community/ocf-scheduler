package cf

import (
	"context"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"golang.org/x/oauth2"
)

// CFClient defines the subset of CF API operations used by this application.
type CFClient interface {
	GetApp(ctx context.Context, guid string) (*resource.App, error)
	CreateTask(ctx context.Context, appGUID string, r *resource.TaskCreate) (*resource.Task, error)
	GetTask(ctx context.Context, guid string) (*resource.Task, error)
	CancelTask(ctx context.Context, guid string) (*resource.Task, error)
	ListUsers(ctx context.Context, opts *cfclient.UserListOptions) ([]*resource.User, error)
	ListRoles(ctx context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error)
	GetServiceInstance(ctx context.Context, guid string) (*resource.ServiceInstance, error)

	// Token returns an access token for the client's own UAA credentials.
	Token() (*oauth2.Token, error)
}
