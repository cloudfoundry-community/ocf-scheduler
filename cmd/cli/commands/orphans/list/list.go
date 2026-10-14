package list

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

// orphanSchedule is a schedule whose job or call is gone, named with the job
// or call it belonged to.
type orphanSchedule struct {
	GUID       string `json:"guid"`
	Enabled    bool   `json:"enabled"`
	Expression string `json:"expression"`
	RefType    string `json:"ref_type"`
	RefGUID    string `json:"ref_guid"`
}

var Command = &cobra.Command{
	Use:   "list",
	Short: "List jobs and calls whose app is gone, and schedules whose job or call is gone",
	Long: `List jobs and calls whose app is gone, and schedules whose job or call is gone

Lists every job and call, in every space, whose app Cloud Controller reports
as not found. The list fails as a whole if Cloud Controller gives any other
answer for any app.

It then lists every schedule whose job or call is no longer stored, such as
one left behind when a delete could not finish.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 0 {
			cmd.Help()
			return fmt.Errorf("\nTakes no arguments")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		response, err := core.Client.Fetch(http.MethodGet, "orphans", nil)
		if err != nil {
			return err
		}

		if !response.Okay() {
			return response.Error()
		}

		// A scheduler too old to list orphan schedules leaves the field
		// out, and then OrphanSchedules stays nil.
		data := struct {
			Resources       []*orphan         `json:"resources"`
			OrphanSchedules []*orphanSchedule `json:"orphan_schedules"`
		}{}

		err = json.Unmarshal(response.Body, &data)
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()
		printOrphans(out, data.Resources)

		if data.OrphanSchedules != nil {
			fmt.Fprintln(out)
			printOrphanSchedules(out, data.OrphanSchedules)
		}

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func printOrphans(out io.Writer, orphans []*orphan) {
	if len(orphans) == 0 {
		fmt.Fprintln(out, "No jobs or calls whose app is gone")
		return
	}

	writer := tabwriter.NewWriter(out, 0, 0, 3, ' ', uint(0))
	fmt.Fprintln(writer, "Type\tName\tGUID\tApp\tSpace\tSchedules")
	fmt.Fprintln(writer, "====\t====\t====\t===\t=====\t=========")

	for _, item := range orphans {
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
}

func printOrphanSchedules(out io.Writer, schedules []*orphanSchedule) {
	if len(schedules) == 0 {
		fmt.Fprintln(out, "No schedules whose job or call is gone")
		return
	}

	fmt.Fprintln(out, "Schedules whose job or call is gone")

	writer := tabwriter.NewWriter(out, 0, 0, 3, ' ', uint(0))
	fmt.Fprintln(writer, "GUID\tFor\tFor GUID\tExpression\tEnabled")
	fmt.Fprintln(writer, "====\t===\t========\t==========\t=======")

	for _, item := range schedules {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%t\n", item.GUID, item.RefType, item.RefGUID, item.Expression, item.Enabled)
	}

	writer.Flush()
}
