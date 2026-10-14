package execute

import (
	"encoding/json"
	"fmt"

	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "execute <job GUID>",
	Short: "Execute a job",
	Long:  `Execute a job`,
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

		execution, err := executeJob(core.Client, jobGUID)
		if err != nil {
			return err
		}

		fmt.Printf(
			"Executed job %s (%s) [Execution GUID: %s]\n",
			job.Name,
			job.GUID,
			execution.GUID,
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

func executeJob(driver *core.Driver, jobGUID string) (*execution, error) {
	response := driver.Post("jobs/"+jobGUID+"/execute", nil, nil)

	if !response.Okay() {
		return nil, response.Action("couldn't execute that job")
	}

	exec := &execution{}

	err := json.Unmarshal(response.Data(), exec)
	if err != nil {
		return nil, err
	}

	return exec, nil
}

type execution struct {
	GUID string `json:"guid"`
}
