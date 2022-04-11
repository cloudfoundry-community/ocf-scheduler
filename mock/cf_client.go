package mock

import (
	"fmt"
	"math/rand"
	"net/url"
	"sync"
	"time"

	cf "github.com/cloudfoundry-community/go-cfclient"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

const (
	dummyGUID = "user-omg-123"
	spaceGUID = "sector-42-a-19"
)

var MaxGetTaskRetries = 10

var spaceManager = cf.V3Role{
	GUID: "j4m3s-t-k1rk",
	Type: "space_manager",
	Relationships: map[string]cf.V3ToOneRelationship{
		"user": cf.V3ToOneRelationship{
			Data: cf.V3Relationship{
				GUID: dummyGUID,
			},
		},

		"space": cf.V3ToOneRelationship{
			Data: cf.V3Relationship{
				GUID: spaceGUID,
			},
		},
	},
}

var spaceDeveloper = cf.V3Role{
	GUID: "g30rg3-luc45",
	Type: "space_developer",
	Relationships: map[string]cf.V3ToOneRelationship{
		"user": cf.V3ToOneRelationship{
			Data: cf.V3Relationship{
				GUID: dummyGUID,
			},
		},

		"space": cf.V3ToOneRelationship{
			Data: cf.V3Relationship{
				GUID: spaceGUID,
			},
		},
	},
}

// Client is a mock of a real *cf.Client instance that implements the
// pieces of the upstream API that we care about as defined by cf.Client
type CFClient struct {
	apps       map[string]cf.App
	tasks      map[string]cf.Task
	retries    map[string]int
	maxretries map[string]int
	locker     sync.Mutex
}

func NewCFClient() (*CFClient, error) {
	client := &CFClient{}
	client.Reset()

	return client, nil
}

func (client *CFClient) AppByGuid(guid string) (cf.App, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	return client.prepareApp(guid, ""), nil
}

func (client *CFClient) CreateTask(request cf.TaskRequest) (cf.Task, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	guid, _ := core.GenGUID()

	task := cf.Task{
		GUID:       guid,
		Command:    request.Command,
		MemoryInMb: request.MemoryInMegabyte,
		DiskInMb:   request.DiskInMegabyte,
		State:      "RUNNING",
	}

	client.tasks[guid] = task
	client.maxretries[guid] = rand.Intn(MaxGetTaskRetries)
	client.retries[guid] = 0

	return task, nil
}

func (client *CFClient) succeed(task cf.Task) cf.Task {
	return cf.Task{
		GUID:       task.GUID,
		Command:    task.Command,
		MemoryInMb: task.MemoryInMb,
		DiskInMb:   task.DiskInMb,
		State:      "SUCCEEDED",
	}
}

func (client *CFClient) fail(task cf.Task) cf.Task {
	return cf.Task{
		GUID:       task.GUID,
		Command:    task.Command,
		MemoryInMb: task.MemoryInMb,
		DiskInMb:   task.DiskInMb,
		State:      "FAILED",
	}
}

func (client *CFClient) GetTaskByGuid(guid string) (cf.Task, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	original, found := client.tasks[guid]
	if !found {
		return cf.Task{}, fmt.Errorf("Task not found")
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

func (client *CFClient) ListUsersByQuery(query url.Values) (cf.Users, error) {
	if query.Get("username") != "dummy" {
		return cf.Users{}, fmt.Errorf("no")
	}

	users := cf.Users{
		cf.User{
			Guid:             dummyGUID,
			CreatedAt:        time.Now().UTC().String(),
			UpdatedAt:        time.Now().UTC().String(),
			Admin:            false,
			Active:           true,
			DefaultSpaceGUID: "space-f1n4l-fr0nt13r",
			Username:         "dummy",
		},
	}

	for i, u := range users {
		fmt.Println("FAKE CLIENT user", i, "guid:", u.Guid)
	}

	return users, nil
}

func (client *CFClient) ListV3RolesByQuery(query url.Values) ([]cf.V3Role, error) {
	output := make([]cf.V3Role, 0)

	if query.Get("user_guids") != dummyGUID {
		return output, fmt.Errorf("no such user")
	}

	//if query.Get("space_guids") != spaceGUID {
	//return output, fmt.Errorf("no such space")
	//}

	output = append(output, spaceManager)
	output = append(output, spaceDeveloper)

	return output, nil
}

func (client *CFClient) Reset() {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.apps = make(map[string]cf.App)
	client.tasks = make(map[string]cf.Task)
	client.retries = make(map[string]int)
	client.maxretries = make(map[string]int)
}

func (client *CFClient) PrepareApp(appGUID string, spaceGUID string) cf.App {
	client.locker.Lock()
	defer client.locker.Unlock()

	return client.prepareApp(appGUID, spaceGUID)
}

func (client *CFClient) prepareApp(appGUID string, spaceGUID string) cf.App {
	// Always retrun the known app if we know it
	if candidate, found := client.apps[appGUID]; found {
		return candidate
	}

	// Generate a space guid if we don't actually receive one
	if len(spaceGUID) == 0 {
		spaceGUID, _ = core.GenGUID()
	}

	output := cf.App{Guid: appGUID, SpaceGuid: spaceGUID}

	client.apps[appGUID] = output

	return output
}

func init() {
	rand.Seed(time.Now().Unix())
}
