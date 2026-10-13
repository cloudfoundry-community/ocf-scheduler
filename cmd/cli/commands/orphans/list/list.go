package list

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

type orphan struct {
	GUID      string                `json:"guid"`
	Type      string                `json:"type"`
	Name      string                `json:"name"`
	AppGUID   string                `json:"app_guid"`
	SpaceGUID string                `json:"space_guid"`
	Schedules []*scheduler.Schedule `json:"schedules"`
}

var Command = &cobra.Command{
	Use:   "list",
	Short: "List jobs and calls whose app is gone",
	Long: `List jobs and calls whose app is gone

Lists every job and call, in every space, whose app Cloud Controller reports
as not found. The list fails as a whole if Cloud Controller gives any other
answer for any app.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			cmd.Help()
			return fmt.Errorf("\nTakes no arguments")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		response := core.Client.Get("orphans", nil)

		if !response.Okay() {
			return response.Error()
		}

		data := struct {
			Resources []*orphan `json:"resources"`
		}{}

		err := json.Unmarshal(response.Data(), &data)
		if err != nil {
			return err
		}

		if len(data.Resources) == 0 {
			fmt.Println("No jobs or calls whose app is gone")
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "Type\tName\tGUID\tApp\tSpace\tSchedules")
		fmt.Fprintln(writer, "====\t====\t====\t===\t=====\t=========")

		for _, item := range data.Resources {
			enabled := 0
			for _, schedule := range item.Schedules {
				if schedule.Enabled {
					enabled++
				}
			}

			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\t%s\t%d (%d enabled)\n",
				item.Type,
				item.Name,
				item.GUID,
				item.AppGUID,
				item.SpaceGUID,
				len(item.Schedules),
				enabled,
			)
		}

		writer.Flush()

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}
