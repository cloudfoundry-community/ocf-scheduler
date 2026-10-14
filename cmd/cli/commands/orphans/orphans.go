package orphans

import (
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/orphans/del"
	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/commands/orphans/list"
)

var Command = &cobra.Command{
	Use:   "orphans",
	Short: "Admin ops for jobs and calls whose app is gone",
	Long: `Admin ops for jobs and calls whose app is gone

These commands find and delete jobs and calls whose app Cloud Controller
reports as not found, and schedules whose job or call is no longer stored.
They need a token with cloud_controller.admin. Please see the Available
Commands section below.`,
}

func init() {
	Command.AddCommand(list.Command)
	Command.AddCommand(del.Command)
}
