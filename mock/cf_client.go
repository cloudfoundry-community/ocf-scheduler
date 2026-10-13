package mock

import (
	"context"
	"fmt"
	"math/rand"
	"slices"
	"sync"

	cfclient "github.com/cloudfoundry/go-cfclient/v3/client"
	"github.com/cloudfoundry/go-cfclient/v3/resource"

	localcf "github.com/cloudfoundry-community/ocf-scheduler/cf"
	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

// Verify CFClient implements the interface at compile time.
var _ localcf.CFClient = (*CFClient)(nil)

const (
	dummyGUID = "user-omg-123"
	spaceGUID = "sector-42-a-19"
)

var MaxGetTaskRetries = 10

var spaceManager = &resource.Role{
	Type: "space_manager",
	Relationships: resource.RoleSpaceUserOrganizationRelationships{
		User: resource.ToOneRelationship{
			Data: &resource.Relationship{
				GUID: dummyGUID,
			},
		},
		Space: resource.ToOneRelationship{
			Data: &resource.Relationship{
				GUID: spaceGUID,
			},
		},
	},
	Resource: resource.Resource{
		GUID: "j4m3s-t-k1rk",
	},
}

var spaceDeveloper = &resource.Role{
	Type: "space_developer",
	Relationships: resource.RoleSpaceUserOrganizationRelationships{
		User: resource.ToOneRelationship{
			Data: &resource.Relationship{
				GUID: dummyGUID,
			},
		},
		Space: resource.ToOneRelationship{
			Data: &resource.Relationship{
				GUID: spaceGUID,
			},
		},
	},
	Resource: resource.Resource{
		GUID: "g30rg3-luc45",
	},
}

// CFClient is a mock of a real CF client that implements the CFClient interface.
type CFClient struct {
	apps       map[string]*resource.App
	tasks      map[string]*resource.Task
	retries    map[string]int
	maxretries map[string]int
	roles      []*resource.Role
	failRoles  bool
	instances  map[string]bool
	failLookup bool
	failApps   bool
	goneApps   map[string]bool
	appErrors  map[string]error
	locker     sync.Mutex
}

func NewCFClient() (*CFClient, error) {
	client := &CFClient{}
	client.Reset()

	return client, nil
}

func (client *CFClient) GetApp(_ context.Context, guid string) (*resource.App, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	if client.failApps {
		return nil, fmt.Errorf("app lookup failed")
	}

	if err, found := client.appErrors[guid]; found {
		return nil, err
	}

	if client.goneApps[guid] {
		return nil, resource.NewResourceNotFoundError()
	}

	return client.prepareApp(guid, ""), nil
}

func (client *CFClient) CreateTask(_ context.Context, appGUID string, req *resource.TaskCreate) (*resource.Task, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	guid, _ := core.GenGUID()

	cmd := ""
	if req.Command != nil {
		cmd = *req.Command
	}
	memMb := 0
	if req.MemoryInMB != nil {
		memMb = *req.MemoryInMB
	}
	diskMb := 0
	if req.DiskInMB != nil {
		diskMb = *req.DiskInMB
	}

	task := &resource.Task{
		Command:    cmd,
		MemoryInMB: memMb,
		DiskInMB:   diskMb,
		State:      "RUNNING",
		Resource: resource.Resource{
			GUID: guid,
		},
	}

	client.tasks[guid] = task
	client.maxretries[guid] = rand.Intn(MaxGetTaskRetries)
	client.retries[guid] = 0

	return task, nil
}

func (client *CFClient) succeed(task *resource.Task) *resource.Task {
	return &resource.Task{
		Command:    task.Command,
		MemoryInMB: task.MemoryInMB,
		DiskInMB:   task.DiskInMB,
		State:      "SUCCEEDED",
		Resource: resource.Resource{
			GUID: task.GUID,
		},
	}
}

func (client *CFClient) fail(task *resource.Task) *resource.Task {
	return &resource.Task{
		Command:    task.Command,
		MemoryInMB: task.MemoryInMB,
		DiskInMB:   task.DiskInMB,
		State:      "FAILED",
		Resource: resource.Resource{
			GUID: task.GUID,
		},
	}
}

func (client *CFClient) GetTask(_ context.Context, guid string) (*resource.Task, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	original, found := client.tasks[guid]
	if !found {
		return nil, fmt.Errorf("Task not found")
	}

	retry, found := client.retries[guid]
	if !found {
		return original, nil
	}

	max := client.maxretries[guid]

	// Since the RunService that uses this method does so in a periodic poll,
	// let's let life imitate art and make it retry several times ;)
	if retry >= max {
		delete(client.retries, guid)
		delete(client.maxretries, guid)
		delete(client.tasks, guid)

		client.tasks[guid] = client.succeed(original)

		return client.tasks[guid], nil
	}

	client.retries[guid] = client.retries[guid] + 1

	return original, nil
}

func (client *CFClient) ListUsers(_ context.Context, opts *cfclient.UserListOptions) ([]*resource.User, error) {
	username := ""
	if opts != nil && len(opts.UserNames.Values) > 0 {
		username = opts.UserNames.Values[0]
	}

	if username != "dummy" {
		return nil, fmt.Errorf("no")
	}

	dummyUsername := "dummy"
	users := []*resource.User{
		{
			Username: &dummyUsername,
			Resource: resource.Resource{
				GUID: dummyGUID,
			},
		},
	}

	return users, nil
}

// ListRoles returns the stock roles for the dummy user plus any granted with
// GrantRole, filtered by the user, space, and type options the real API
// honors.
func (client *CFClient) ListRoles(_ context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	if client.failRoles {
		return nil, fmt.Errorf("role lookup failed")
	}

	candidates := append([]*resource.Role{spaceManager, spaceDeveloper}, client.roles...)
	output := make([]*resource.Role, 0)

	for _, role := range candidates {
		if opts != nil {
			if !matches(opts.UserGUIDs.Values, relationshipGUID(role.Relationships.User)) ||
				!matches(opts.SpaceGUIDs.Values, relationshipGUID(role.Relationships.Space)) ||
				!matches(opts.Types.Values, role.Type) {
				continue
			}
		}

		output = append(output, role)
	}

	return output, nil
}

// GrantRole gives a user a space role that ListRoles then reports.
func (client *CFClient) GrantRole(userGUID, spaceGUID, roleType string) {
	client.locker.Lock()
	defer client.locker.Unlock()

	guid, _ := core.GenGUID()

	client.roles = append(client.roles, &resource.Role{
		Type: roleType,
		Relationships: resource.RoleSpaceUserOrganizationRelationships{
			User:  resource.ToOneRelationship{Data: &resource.Relationship{GUID: userGUID}},
			Space: resource.ToOneRelationship{Data: &resource.Relationship{GUID: spaceGUID}},
		},
		Resource: resource.Resource{GUID: guid},
	})
}

// FailRoleLookups makes ListRoles return an error, as an unreachable CF API
// would.
func (client *CFClient) FailRoleLookups(fail bool) {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.failRoles = fail
}

func matches(filter []string, value string) bool {
	return len(filter) == 0 || slices.Contains(filter, value)
}

func relationshipGUID(relationship resource.ToOneRelationship) string {
	if relationship.Data == nil {
		return ""
	}

	return relationship.Data.GUID
}

func (client *CFClient) Reset() {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.apps = make(map[string]*resource.App)
	client.tasks = make(map[string]*resource.Task)
	client.retries = make(map[string]int)
	client.maxretries = make(map[string]int)
	client.roles = nil
	client.failRoles = false
	client.instances = make(map[string]bool)
	client.failLookup = false
	client.failApps = false
	client.goneApps = make(map[string]bool)
	client.appErrors = make(map[string]error)
}

// GetServiceInstance knows the instances added with PrepareServiceInstance
// and answers CF-ResourceNotFound for any other GUID.
func (client *CFClient) GetServiceInstance(_ context.Context, guid string) (*resource.ServiceInstance, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	if client.failLookup {
		return nil, fmt.Errorf("service instance lookup failed")
	}

	if !client.instances[guid] {
		return nil, resource.NewResourceNotFoundError()
	}

	return &resource.ServiceInstance{Resource: resource.Resource{GUID: guid}}, nil
}

// PrepareServiceInstance makes Cloud Controller know a service instance GUID.
func (client *CFClient) PrepareServiceInstance(guid string) {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.instances[guid] = true
}

// FailServiceInstanceLookups makes GetServiceInstance fail as an unreachable
// CF API would.
func (client *CFClient) FailServiceInstanceLookups(fail bool) {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.failLookup = fail
}

// FailAppLookups makes GetApp fail as an unreachable CF API would.
func (client *CFClient) FailAppLookups(fail bool) {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.failApps = fail
}

// RemoveApp makes GetApp answer CF-ResourceNotFound for appGUID, as Cloud
// Controller does once the app is deleted.
func (client *CFClient) RemoveApp(appGUID string) {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.goneApps[appGUID] = true
}

// RestoreApp undoes RemoveApp, so GetApp finds appGUID again.
func (client *CFClient) RestoreApp(appGUID string) {
	client.locker.Lock()
	defer client.locker.Unlock()

	delete(client.goneApps, appGUID)
}

// FailAppLookup makes GetApp answer err for appGUID alone, or clears that
// failure when err is nil.
func (client *CFClient) FailAppLookup(appGUID string, err error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	if err == nil {
		delete(client.appErrors, appGUID)
		return
	}

	client.appErrors[appGUID] = err
}

func (client *CFClient) PrepareApp(appGUID string, spaceGUID string) *resource.App {
	client.locker.Lock()
	defer client.locker.Unlock()

	return client.prepareApp(appGUID, spaceGUID)
}

func (client *CFClient) prepareApp(appGUID string, spGUID string) *resource.App {
	// Always return the known app if we know it
	if candidate, found := client.apps[appGUID]; found {
		return candidate
	}

	// Generate a space guid if we don't actually receive one
	if len(spGUID) == 0 {
		spGUID, _ = core.GenGUID()
	}

	output := &resource.App{
		Relationships: resource.AppRelationships{
			Space: resource.ToOneRelationship{
				Data: &resource.Relationship{GUID: spGUID},
			},
		},
		Resource: resource.Resource{
			GUID: appGUID,
		},
	}

	client.apps[appGUID] = output

	return output
}
