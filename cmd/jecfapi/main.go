package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/cloudfoundry/go-cfclient/v3/resource"
	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

func Server(bind string, cfURL string, uaaURL string) *http.Server {
	client, _ := mock.NewCFClient()
	e := echo.New()

	e.GET("/v2/info", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			map[string]any{
				"authorization_endpoint":       uaaURL,
				"token_endpoint":               uaaURL,
				"logging_endpoint":             cfURL,
				"name":                         "",
				"build":                        "",
				"support":                      "https://support.example.com",
				"version":                      0,
				"description":                  "",
				"min_cli_version":              "6.23.0",
				"min_recommended_cli_version":  "6.23.0",
				"api_version":                  "2.103.0",
				"app_ssh_endpoint":             "ssh.example.com:2222",
				"app_ssh_host_key_fingerprint": "00:00:00:00:00:00:00:00:00:00:00:00:00:00:00:01",
				"app_ssh_oauth_client":         "ssh-proxy",
				"doppler_logging_endpoint":     "wss://doppler.example.com:443",
				"routing_endpoint":             "https://api.example.com/routing",
			},
		)
	})

	// Also serve the root endpoint for v3 client discovery
	e.GET("/", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			map[string]any{
				"links": map[string]any{
					"self":          map[string]string{"href": cfURL},
					"cloud_controller_v3": map[string]string{"href": cfURL + "/v3"},
					"uaa":           map[string]string{"href": uaaURL},
					"login":         map[string]string{"href": uaaURL},
				},
			},
		)
	})

	// GetApp (v3)
	e.GET("/v3/apps/:guid", func(c echo.Context) error {
		app := client.PrepareApp(c.Param("guid"), "")
		return c.JSON(http.StatusOK, app)
	})

	// CreateTask
	e.POST("/v3/apps/:guid/tasks", func(c echo.Context) error {
		client.PrepareApp(c.Param("guid"), "")
		input := make(map[string]any)

		if err := c.Bind(&input); err != nil {
			fmt.Println("task request unmarshal error:", err)
			return c.JSON(http.StatusUnprocessableEntity, []string{"lolwut"})
		}

		disk := 0
		if v, ok := input["disk_in_mb"]; ok {
			switch d := v.(type) {
			case float64:
				disk = int(d)
			case string:
				disk, _ = strconv.Atoi(d)
			}
		}

		mem := 0
		if v, ok := input["memory_in_mb"]; ok {
			switch m := v.(type) {
			case float64:
				mem = int(m)
			case string:
				mem, _ = strconv.Atoi(m)
			}
		}

		log_rate := 0
		if v, ok := input["log_rate_in_bps"]; ok {
			switch d := v.(type) {
			case float64:
				log_rate = int(d)
			case string:
				log_rate, _ = strconv.Atoi(d)
			}
		}

		cmd := ""
		if v, ok := input["command"].(string); ok {
			cmd = v
		}

		realReq := resource.NewTaskCreateWithCommand(cmd)
		realReq.WithDiskInMB(disk).WithMemoryInMB(mem).WithLogRateLimitInBytesPerSecond(log_rate)

		task, err := client.CreateTask(context.Background(), c.Param("guid"), realReq)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		return c.JSON(
			http.StatusCreated,
			task,
		)
	})

	// GetTaskByGuid
	e.GET("/v3/tasks/:guid", func(c echo.Context) error {
		task, err := client.GetTask(context.Background(), c.Param("guid"))
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		return c.JSON(
			http.StatusOK,
			task,
		)
	})

	// ListUsers (v3)
	e.GET("/v3/users", func(c echo.Context) error {
		username := c.QueryParam("usernames")

		// Build a mock response in v3 format
		users := make([]*resource.User, 0)

		if len(username) > 0 {
			// Use the mock client to get users
			mockUsers, err := client.ListUsers(context.Background(), nil)
			if err == nil {
				for _, u := range mockUsers {
					if u.Username == username {
						users = append(users, u)
					}
				}
			}
		}

		output := &resource.UserList{
			Pagination: resource.Pagination{
				TotalPages:   1,
				TotalResults: len(users),
			},
			Resources: users,
		}

		return c.JSON(http.StatusOK, output)
	})

	// ListRoles (v3)
	e.GET("/v3/roles", func(c echo.Context) error {
		userGUIDs := c.QueryParam("user_guids")

		roles := make([]*resource.Role, 0)

		if len(userGUIDs) > 0 {
			mockRoles, err := client.ListRoles(context.Background(), nil)
			if err == nil {
				for _, r := range mockRoles {
					if r.Relationships.User.Data != nil && r.Relationships.User.Data.GUID == userGUIDs {
						roles = append(roles, r)
					}
				}
			}
		}

		output := &resource.RoleList{
			Pagination: resource.Pagination{
				TotalPages:   1,
				TotalResults: len(roles),
			},
			Resources: roles,
		}

		return c.JSON(http.StatusOK, output)
	})

	e.GET("*", func(c echo.Context) error {
		fmt.Println("Got a GET request I didn't recognize:", c.Request().URL)

		return c.JSON(http.StatusInternalServerError, "")
	})

	e.POST("*", func(c echo.Context) error {
		fmt.Println("Got a POST request I didn't recognize:", c.Request().URL)

		return c.JSON(http.StatusInternalServerError, "")
	})

	server := e.Server
	server.Addr = bind

	return server
}

func main() {
	cfURL := os.Getenv("CF_ENDPOINT")
	if len(cfURL) == 0 {
		fmt.Println("CF_ENDPOINT not set")
		os.Exit(1)
	}

	uaaURL := os.Getenv("UAA_ENDPOINT")
	if len(uaaURL) == 0 {
		fmt.Println("UAA_ENDPOINT not set")
		os.Exit(1)
	}

	server := Server("0.0.0.0:8002", cfURL, uaaURL)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("stopping the server")
		}
	}()

	fmt.Printf("listening for connections on %s\n", server.Addr)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		fmt.Println(err.Error())
		os.Exit(2)
	}
}
