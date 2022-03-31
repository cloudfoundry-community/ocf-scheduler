package main

import (
	"fmt"
	"os"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

func main() {
	// set up the global http driver
	driver, err := core.NewDriver("http://localhost:8000", "jeremy")
	if err != nil {
		fmt.Println("Oh no! I couldn't create a http client!")
		os.Exit(1)
	}

	core.Client = driver

	if commands.Execute() != nil {
		os.Exit(1)
	}
}
