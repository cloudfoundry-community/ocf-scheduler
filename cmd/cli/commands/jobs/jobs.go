package jobs

import (
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/all"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/create"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/del"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/execute"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/history"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/jobs/schedules"
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
	Command.AddCommand(del.Command)
	Command.AddCommand(history.Command)
	Command.AddCommand(schedules.Command)
}
