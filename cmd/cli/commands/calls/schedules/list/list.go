package list

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "list <call GUID>",
	Short: "List all schedules for a call",
	Long:  `List all schedules for a call`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <call GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]

		call, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		schedules := getCallSchedules(core.Client, callGUID)
		if len(schedules) == 0 {
			fmt.Println("No schedules for call", callGUID)
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "Call Name\tSchedule\tWhen")
		fmt.Fprintln(writer, "=========\t========\t====")

		for _, schedule := range schedules {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", call.Name, schedule.GUID, schedule.Expression)
		}

		writer.Flush()

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func getCall(driver *core.Driver, callGUID string) (*scheduler.Call, error) {
	response := driver.Get("calls/"+callGUID, nil)

	if !response.Okay() {
		return nil, response.Error()
	}

	call := &scheduler.Call{}

	err := json.Unmarshal(response.Data(), call)
	if err != nil {
		return nil, err
	}

	return call, nil
}

func getCallSchedules(driver *core.Driver, callGUID string) []*scheduler.Schedule {
	dumb := make([]*scheduler.Schedule, 0)

	response := core.Client.Get("calls/"+callGUID+"/schedules", nil)

	if !response.Okay() {
		return dumb
	}

	data := struct {
		Resources []*scheduler.Schedule `json:"resources"`
	}{}

	err := json.Unmarshal(response.Data(), &data)
	if err != nil {
		return dumb
	}

	return data.Resources
}
