package create

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
	Use:   "create <call GUID> <cron expression>",
	Short: "Create a schedule for a call",
	Long:  `Create a schedule for a call`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <call GUID> <cron expression>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]
		cronExpression := args[1]

		call, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		schedule, err := scheduleCall(core.Client, callGUID, cronExpression)
		if err != nil {
			return fmt.Errorf("couldn't schedule that call")
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "Call Name\tSchedule\tWhen")
		fmt.Fprintln(writer, "=========\t========\t====")

		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\nn",
			call.Name,
			schedule.GUID,
			schedule.Expression,
		)

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

func scheduleCall(driver *core.Driver, callGUID string, when string) (*scheduler.Schedule, error) {

	schedule := &schedulePayload{
		Enabled:        true,
		Expression:     when,
		ExpressionType: "cron_expression",
	}

	sdata, err := json.Marshal(schedule)
	if err != nil {
		return nil, err
	}

	response := driver.Post("calls/"+callGUID+"/schedules", nil, sdata)

	if !response.Okay() {
		return nil, response.Error()
	}

	output := &scheduler.Schedule{}

	err = json.Unmarshal(response.Data(), output)
	if err != nil {
		return nil, err
	}

	return output, nil
}

type schedulePayload struct {
	Enabled        bool   `json:"enabled"`
	Expression     string `json:"expression"`
	ExpressionType string `json:"expression_type"`
}
