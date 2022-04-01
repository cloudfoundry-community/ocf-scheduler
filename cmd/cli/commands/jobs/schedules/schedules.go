package schedules

import (
	"github.com/spf13/cobra"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/schedules/create"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/schedules/del"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/schedules/history"
	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/commands/jobs/schedules/list"
)

var Command = &cobra.Command{
	Use:   "schedules",
	Short: "Ops for job schedules",
	Long: `Ops for job schedules

This is just a collection of job-related schedule ops commands. Please see the
Available Commands section below.`,
}

func init() {
	Command.AddCommand(create.Command)
	Command.AddCommand(list.Command)
	Command.AddCommand(del.Command)
	Command.AddCommand(history.Command)
}
