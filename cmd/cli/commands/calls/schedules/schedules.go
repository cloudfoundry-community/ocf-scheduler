package schedules

import (
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/schedules/create"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/schedules/del"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/schedules/history"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/calls/schedules/list"
)

var Command = &cobra.Command{
	Use:   "schedules",
	Short: "Ops for call schedules",
	Long: `Ops for call schedules

This is just a collection of call-related schedule ops commands. Please see the
Available Commands section below.`,
}

func init() {
	Command.AddCommand(create.Command)
	Command.AddCommand(list.Command)
	Command.AddCommand(del.Command)
	Command.AddCommand(history.Command)
}
