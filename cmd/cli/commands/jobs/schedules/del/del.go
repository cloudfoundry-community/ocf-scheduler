package del

import (
	"encoding/json"
	"fmt"

	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
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
			return err
		}

		err = deleteSchedule(core.Client, job.GUID, scheduleGUID)
		if err != nil {
			return err
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
		return nil, response.Lookup("job")
	}

	job := &scheduler.Job{}

	err := json.Unmarshal(response.Data(), job)
	if err != nil {
		return nil, fmt.Errorf("couldn't look up that job: %w", err)
	}

	return job, nil
}

func deleteSchedule(driver *core.Driver, jobGUID string, scheduleGUID string) error {
	response := core.Client.Delete(
		"jobs/"+jobGUID+"/schedules/"+scheduleGUID,
		nil,
	)

	if !response.Okay() {
		return response.Action(fmt.Sprintf("couldn't delete schedule %s", scheduleGUID))
	}

	return nil
}
