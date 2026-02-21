package cf

import (
	"context"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
)

// RealCFClient wraps the official go-cfclient v3 client
// and implements the CFClient interface.
type RealCFClient struct {
	inner *cfclient.Client
}

func NewRealCFClient(c *cfclient.Client) *RealCFClient {
	return &RealCFClient{inner: c}
}

func (r *RealCFClient) GetApp(ctx context.Context, guid string) (*resource.App, error) {
	return r.inner.Applications.Get(ctx, guid)
}

func (r *RealCFClient) CreateTask(ctx context.Context, appGUID string, req *resource.TaskCreate) (*resource.Task, error) {
	return r.inner.Tasks.Create(ctx, appGUID, req)
}

func (r *RealCFClient) GetTask(ctx context.Context, guid string) (*resource.Task, error) {
	return r.inner.Tasks.Get(ctx, guid)
}

func (r *RealCFClient) ListUsers(ctx context.Context, opts *cfclient.UserListOptions) ([]*resource.User, error) {
	return r.inner.Users.ListAll(ctx, opts)
}

func (r *RealCFClient) ListRoles(ctx context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error) {
	return r.inner.Roles.ListAll(ctx, opts)
}
