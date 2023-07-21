package main

import (
	"fmt"
	"os"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

func main() {
	schedulerURL := os.Getenv("SCHEDULER_URL")
	if len(schedulerURL) == 0 {
		fmt.Println("SCHEDULER_URL is required")
		os.Exit(1)
	}

	schedulerToken := os.Getenv("SCHEDULER_TOKEN")
	if len(schedulerToken) == 0 {
		fmt.Println("SCHEDULER_TOKEN is required")
		os.Exit(1)
	}

	// set up the global http driver
	driver, err := core.NewDriver(schedulerURL, schedulerToken)
	if err != nil {
		fmt.Println("Oh no! I couldn't create a http client!")
		os.Exit(1)
	}

	core.Client = driver

	if commands.Execute() != nil {
		os.Exit(1)
	}
}
