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
	Use:   "create <job GUID> <cron expression>",
	Short: "Create a schedule for a job",
	Long:  `Create a schedule for a job`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <job GUID> <cron expression>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		jobGUID := args[0]
		cronExpression := args[1]

		job, err := getJob(core.Client, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that job")
		}

		schedule, err := scheduleJob(core.Client, jobGUID, cronExpression)
		if err != nil {
			return fmt.Errorf("couldn't schedule that job")
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "Job Name\tSchedule\tWhen")
		fmt.Fprintln(writer, "========\t========\t====")

		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\nn",
			job.Name,
			schedule.GUID,
			schedule.Expression,
		)

		writer.Flush()

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func getJob(driver *core.Driver, jobGUID string) (*scheduler.Job, error) {
	response := driver.Get("jobs/"+jobGUID, nil)

	if !response.Okay() {
		return nil, response.Error()
	}

	job := &scheduler.Job{}

	err := json.Unmarshal(response.Data(), job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func scheduleJob(driver *core.Driver, jobGUID string, when string) (*scheduler.Schedule, error) {

	schedule := &schedulePayload{
		Enabled:        true,
		Expression:     when,
		ExpressionType: "cron_expression",
	}

	sdata, err := json.Marshal(schedule)
	if err != nil {
		return nil, err
	}

	response := driver.Post("jobs/"+jobGUID+"/schedules", nil, sdata)

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
