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
	Use:   "history <job GUID> <schedule GUID>",
	Short: "Get the scheduled execution history for a job",
	Long:  `Get the scheduled execution history for a job`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <job GUID> <schedule GUID")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		jobGUID := args[0]
		scheduleGUID := args[1]

		job, err := getJob(core.Client, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that job")
		}

		executions := getScheduleExecutions(core.Client, job.GUID, scheduleGUID)

		if len(executions) == 0 {
			fmt.Println("No executions for schedule", scheduleGUID)
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "State\tScheduled Time\tStart Time\tEnd Time")
		fmt.Fprintln(writer, "=====\t==============\t==========\t========")

		for _, exec := range executions {
			fmt.Fprintln(
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

func getScheduleExecutions(driver *core.Driver, jobGUID string, scheduleGUID string) []*execution {
	dumb := make([]*execution, 0)

	response := driver.Get("jobs/"+jobGUID+"/schedules/"+scheduleGUID+"/history", nil)

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
