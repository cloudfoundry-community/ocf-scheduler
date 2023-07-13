package history

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "history <call GUID> <schedule GUID>",
	Short: "Get the scheduled execution history for a call",
	Long:  `Get the scheduled execution history for a call`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <call GUID> <schedule GUID")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]
		scheduleGUID := args[1]

		call, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		executions := getScheduleExecutions(core.Client, call.GUID, scheduleGUID)

		if len(executions) == 0 {
			fmt.Println("No executions for schedule", scheduleGUID)
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "State\tScheduled Time\tStart Time\tEnd Time")
		fmt.Fprintln(writer, "=====\t==============\t==========\t========")

		for _, exec := range executions {
			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\t%s\n",
				exec.State,
				exec.ScheduledTime,
				exec.ExecutionStartTime,
				exec.ExecutionEndTime,
			)
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

func getScheduleExecutions(driver *core.Driver, callGUID string, scheduleGUID string) []*execution {
	dumb := make([]*execution, 0)

	response := driver.Get("calls/"+callGUID+"/schedules/"+scheduleGUID+"/history", nil)

	if !response.Okay() {
		return dumb
	}

	data := struct {
		Resources []*execution `json:"resources"`
	}{}

	err := json.Unmarshal(response.Data(), &data)
	if err != nil {
		return dumb
	}

	return data.Resources
}

type execution struct {
	State              string    `json:"state"`
	ScheduledTime      time.Time `json:"scheduled_time"`
	ExecutionStartTime time.Time `json:"execution_start_time"`
	ExecutionEndTime   time.Time `json:"execution_end_time"`
}
