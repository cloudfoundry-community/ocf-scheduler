package main

import (
	"os"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands"
)

func main() {
	if commands.Execute() != nil {
		os.Exit(1)
	}
}
