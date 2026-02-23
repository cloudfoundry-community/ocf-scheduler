package mock

import (
	"context"
	"fmt"
	"math/rand"
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

func (client *CFClient) ListRoles(_ context.Context, opts *cfclient.RoleListOptions) ([]*resource.Role, error) {
	userGUID := ""
	if opts != nil && len(opts.UserGUIDs.Values) > 0 {
		userGUID = opts.UserGUIDs.Values[0]
	}

	if userGUID != dummyGUID {
		return nil, fmt.Errorf("no such user")
	}

	output := []*resource.Role{spaceManager, spaceDeveloper}
	return output, nil
}

func (client *CFClient) Reset() {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.apps = make(map[string]*resource.App)
	client.tasks = make(map[string]*resource.Task)
	client.retries = make(map[string]int)
	client.maxretries = make(map[string]int)
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
