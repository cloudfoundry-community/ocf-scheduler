package jobs

import (
	"github.com/spf13/cobra"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/all"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/create"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/execute"
)

var Command = &cobra.Command{
	Use:   "jobs",
	Short: "Ops for jobs",
	Long: `Ops for jobs

This is just a collection of job-related ops commands. Please see the
Available Commands section below.`,
}

func init() {
	Command.AddCommand(all.Command)
	Command.AddCommand(create.Command)
	Command.AddCommand(execute.Command)
}
