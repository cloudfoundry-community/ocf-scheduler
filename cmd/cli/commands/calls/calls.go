package calls

import (
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/all"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/create"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/del"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/execute"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/history"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/schedules"
)

var Command = &cobra.Command{
	Use:   "calls",
	Short: "Ops for calls",
	Long: `Ops for calls

This is just a collection of call-related ops commands. Please see the
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
