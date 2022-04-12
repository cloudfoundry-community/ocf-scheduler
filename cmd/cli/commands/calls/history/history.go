package history

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "history <call GUID>",
	Short: "Get a call's execution history",
	Long:  `Get a call's execution history`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <call GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]

		_, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		executions := getCallExecutions(core.Client, callGUID)

		if len(executions) == 0 {
			fmt.Println("No executions for call", callGUID)

			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "State\tExecution Start Time\tExecution End Time")
		fmt.Fprintln(writer, "=====\t====================\t==================")

		for _, execution := range executions {
			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				execution.State,
				execution.ExecutionStartTime,
				execution.ExecutionEndTime,
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

func getCallExecutions(driver *core.Driver, callGUID string) []*execution {
	dumb := make([]*execution, 0)

	response := driver.Get("calls/"+callGUID+"/history", nil)

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
	ExecutionStartTime time.Time `json:"execution_start_time"`
	ExecutionEndTime   time.Time `json:"execution_end_time"`
}
