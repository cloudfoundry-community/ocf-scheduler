package del

import (
	"encoding/json"
	"fmt"

	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
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
			return err
		}

		err = deleteJob(core.Client, jobGUID)
		if err != nil {
			return err
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
		return nil, response.Lookup("job")
	}

	job := &scheduler.Job{}

	err := json.Unmarshal(response.Data(), job)
	if err != nil {
		return nil, fmt.Errorf("couldn't look up that job: %w", err)
	}

	return job, nil
}

func deleteJob(driver *core.Driver, jobGUID string) error {
	response := driver.Delete("jobs/"+jobGUID, nil)

	if !response.Okay() {
		return response.Action("couldn't delete that job")
	}

	return nil
}
