package mock

import (
	"fmt"
	"math/rand"
	"sync"
	"time"

	cf "github.com/cloudfoundry-community/go-cfclient"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

var MaxGetTaskRetries = 30

// Client is a mock of a real *cf.Client instance that implements the
// pieces of the upstream API that we care about as defined by cf.Client
type Client struct {
	apps       map[string]cf.App
	tasks      map[string]cf.Task
	retries    map[string]int
	maxretries map[string]int
	locker     sync.Mutex
}

func NewClient() (*Client, error) {
	client := &Client{}
	client.Reset()

	return client, nil
}

func (client *Client) AppByGuid(guid string) (cf.App, error) {
	client.locker.Lock()
	defer client.locker.Unlock()

	return client.prepareApp(guid, ""), nil
}

func (client *Client) CreateTask(request cf.TaskRequest) (cf.Task, error) {
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

func (client *Client) succeed(task cf.Task) cf.Task {
	return cf.Task{
		GUID:       task.GUID,
		Command:    task.Command,
		MemoryInMb: task.MemoryInMb,
		DiskInMb:   task.DiskInMb,
		State:      "SUCCEEDED",
	}
}

func (client *Client) fail(task cf.Task) cf.Task {
	return cf.Task{
		GUID:       task.GUID,
		Command:    task.Command,
		MemoryInMb: task.MemoryInMb,
		DiskInMb:   task.DiskInMb,
		State:      "FAILED",
	}
}

func (client *Client) GetTaskByGuid(guid string) (cf.Task, error) {
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

func (client *Client) Reset() {
	client.locker.Lock()
	defer client.locker.Unlock()

	client.apps = make(map[string]cf.App)
	client.tasks = make(map[string]cf.Task)
	client.retries = make(map[string]int)
	client.maxretries = make(map[string]int)
}

func (client *Client) PrepareApp(appGUID string, spaceGUID string) cf.App {
	client.locker.Lock()
	defer client.locker.Unlock()

	return client.prepareApp(appGUID, spaceGUID)
}

func (client *Client) prepareApp(appGUID string, spaceGUID string) cf.App {
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
