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
	Use:   "list <job GUID>",
	Short: "List all schedules for a job",
	Long:  `List all schedules for a job`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <job GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		jobGUID := args[0]

		job, err := getJob(core.Client, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that job")
		}

		schedules := getJobSchedules(core.Client, jobGUID)
		if len(schedules) == 0 {
			fmt.Println("No schedules for job", jobGUID)
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "Job Name\tSchedule\tWhen")
		fmt.Fprintln(writer, "========\t========\t====")

		for _, schedule := range schedules {
			fmt.Fprintf(writer, "%s\t%s\t%s\n", job.Name, schedule.GUID, schedule.Expression)
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

func getJobSchedules(driver *core.Driver, jobGUID string) []*scheduler.Schedule {
	dumb := make([]*scheduler.Schedule, 0)

	response := core.Client.Get("jobs/"+jobGUID+"/schedules", nil)

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
