package main

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"time"

	cf "github.com/cloudfoundry-community/go-cfclient"
	"github.com/labstack/echo/v4"

	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

type pageref struct {
	Href string `json:"href"`
}

type pagination struct {
	First        *pageref `json:"first,omitempty"`
	Last         *pageref `json:"last,omitempty"`
	Next         *pageref `json:"next,omitempty"`
	Previous     *pageref `json:"previous,omitempty"`
	TotalPages   int      `json:"total_pages"`
	TotalResults int      `json:"total_results"`
}

type userCollection struct {
	Pagination *pagination `json:"pagination"`
	Resources  []userResp  `json:"resources"`
}

type roleCollection struct {
	Pagination *pagination `json:"pagination"`
	Resources  []cf.V3Role `json:"resources"`
}

func Server(bind string, cfURL string, uaaURL string) *http.Server {
	client, _ := mock.NewCFClient()
	e := echo.New()

	e.GET("/v2/info", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			map[string]interface{}{
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

	// AppByGuid
	e.GET("/v2/apps/:guid", func(c echo.Context) error {
		app := client.PrepareApp(c.Param("guid"), "")

		fmt.Println("app space guid:", app.SpaceGuid)

		wrapped := struct {
			Meta   cf.Meta `json:"metadata"`
			Entity cf.App  `json:"entity"`
		}{
			Meta: cf.Meta{
				Guid:      app.Guid,
				Url:       "",
				CreatedAt: app.CreatedAt,
				UpdatedAt: app.UpdatedAt,
			},

			Entity: app,
		}

		return c.JSON(
			http.StatusOK,
			wrapped,
		)
	})

	// CreateTask
	e.POST("/v3/apps/:guid/tasks", func(c echo.Context) error {
		client.PrepareApp(c.Param("guid"), "")
		input := make(map[string]string)

		if err := c.Bind(&input); err != nil {
			fmt.Println("task request unmarshal error:", err)
			return c.JSON(http.StatusUnprocessableEntity, []string{"lolwut"})
		}

		disk, err := strconv.Atoi(input["disk_in_mb"])
		if err != nil {
			fmt.Println("disk_in_mb doesn't convert to int")
			return c.JSON(http.StatusUnprocessableEntity, []string{"lolwut"})
		}

		mem, err := strconv.Atoi(input["memory_in_mb"])
		if err != nil {
			fmt.Println("memory_in_mb  doesn't convert to int")
			return c.JSON(http.StatusUnprocessableEntity, []string{"lolwut"})
		}

		realReq := cf.TaskRequest{
			Command:          input["command"],
			DiskInMegabyte:   disk,
			MemoryInMegabyte: mem,
		}

		task, err := client.CreateTask(realReq)
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
		task, err := client.GetTaskByGuid(c.Param("guid"))
		if err != nil {
			return c.JSON(http.StatusNotFound, "")
		}

		return c.JSON(
			http.StatusOK,
			task,
		)
	})

	// ListUsersByQuery
	e.GET("/v2/users", func(c echo.Context) error {
		query := url.Values{}
		if username := c.QueryParam("username"); len(username) > 0 {
			query.Add("username", username)
		}

		users, err := client.ListUsersByQuery(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		wrappedUsers := make([]userResp, 0)
		for _, user := range users {
			wrappedUsers = append(
				wrappedUsers,
				userResp{
					Meta: cf.Meta{
						Guid:      user.Guid,
						Url:       "",
						CreatedAt: user.CreatedAt,
						UpdatedAt: user.UpdatedAt,
					},

					Entity: user,
				},
			)
		}

		output := &userCollection{
			Resources: wrappedUsers,
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(users),
				First:        &pageref{Href: "first"},
				Last:         &pageref{Href: "last"},
				Next:         &pageref{Href: "next"},
				Previous:     &pageref{Href: "previous"},
			},
		}

		return c.JSON(
			http.StatusOK,
			output,
		)
	})

	// ListV3RolesByQuery
	e.GET("/v3/roles", func(c echo.Context) error {
		query := url.Values{}
		if userGUIDs := c.QueryParam("user_guids"); len(userGUIDs) > 0 {
			query.Add("user_guids", userGUIDs)
		}

		roles, err := client.ListV3RolesByQuery(query)
		if err != nil {
			return c.JSON(http.StatusInternalServerError, "")
		}

		output := &roleCollection{
			Resources: roles,
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(roles),
				First:        nil,
				Last:         nil,
				Next:         nil,
				Previous:     nil,
			},
		}

		return c.JSON(
			http.StatusOK,
			output,
		)
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

	quit := make(chan os.Signal)
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

type userResp struct {
	Meta   cf.Meta `json:"metadata"`
	Entity cf.User `json:"entity"`
}

type taskReq struct {
	Command string `json:"command"`
	Disk    int    `json:"disk_in_mb"`
	Mem     int    `json:"memory_in_mb"`
}
