package del

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "del <job GUID>",
	Short: "Delete a job",
	Long:  `Delete a job`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <job GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		jobGUID := args[0]

		job, err := getJob(core.Client, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that job")
		}

		err = deleteJob(core.Client, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't delete that job")
		}

		fmt.Printf(
			"Deleted job %s (%s)\n",
			job.Name,
			job.GUID,
		)

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

func deleteJob(driver *core.Driver, jobGUID string) error {
	response := driver.Delete("jobs/"+jobGUID, nil)

	if !response.Okay() {
		return response.Error()
	}

	return nil
}
