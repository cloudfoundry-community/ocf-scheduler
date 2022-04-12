package del

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "del <job GUID> <schedule GUID>",
	Short: "Delete a schedule for a job",
	Long:  `Delete a schedule for a job`,
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

		err = deleteSchedule(core.Client, job.GUID, scheduleGUID)
		if err != nil {
			return fmt.Errorf("couldn't delete schedule %s", scheduleGUID)
		}

		fmt.Println("Schedule", scheduleGUID, "deleted.")
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

func deleteSchedule(driver *core.Driver, jobGUID string, scheduleGUID string) error {
	response := core.Client.Delete(
		"jobs/"+jobGUID+"/schedules/"+scheduleGUID,
		nil,
	)

	if !response.Okay() {
		return response.Error()
	}

	return nil
}
