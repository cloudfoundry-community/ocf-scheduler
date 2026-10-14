package cf

import (
	"context"
	"fmt"
	"sync"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"golang.org/x/oauth2"
)

// RealCFClient wraps the official go-cfclient v3 client
// and implements the CFClient interface.
type RealCFClient struct {
	inner *cfclient.Client

	lock   sync.Mutex
	tokens oauth2.TokenSource
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

func (r *RealCFClient) CancelTask(ctx context.Context, guid string) (*resource.Task, error) {
	return r.inner.Tasks.Cancel(ctx, guid)
}

func (r *RealCFClient) ListUsers(ctx context.Context, opts *cfclient.UserListOptions) ([]*resource.User, error) {
	return r.inner.Users.ListAll(ctx, opts)
}

func (r *RealCFClient) GetServiceInstance(ctx context.Context, guid string) (*resource.ServiceInstance, error) {
	return r.inner.ServiceInstances.Get(ctx, guid)
}

func (r *RealCFClient) ListRoles(ctx context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error) {
	return r.inner.Roles.ListAll(ctx, opts)
}

// Token returns an access token for the client's own credentials. It asks
// UAA with the same config, and so the same grant and requested scopes, that
// the client uses for its Cloud Controller calls, and it reuses the token
// until the token expires.
func (r *RealCFClient) Token() (*oauth2.Token, error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.tokens == nil {
		src, err := r.inner.CreateOAuth2TokenSource(context.Background())
		if err != nil {
			return nil, fmt.Errorf("could not create a token source: %w", err)
		}
		r.tokens = oauth2.ReuseTokenSource(nil, src)
	}

	return r.tokens.Token()
}
