package execute

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/scheduler-for-ocf/core"
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

		driver, err := hype.New("http://localhost:8000")
		if err != nil {
			return fmt.Errorf("couldn't hype it up: %s", err.Error())
		}

		job, err := getJob(driver, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that job")
		}

		execution, err := executeJob(driver, jobGUID)
		if err != nil {
			return fmt.Errorf("couldn't execute that job")
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

func getJob(driver *hype.Driver, jobGUID string) (*core.Job, error) {
	response := driver.
		Get("jobs/"+jobGUID, nil).
		WithHeaderSet(
			hype.NewHeader("Authorization", "jeremy"),
			hype.NewHeader("Accept", "application/json"),
			hype.NewHeader("Content-Type", "application/json"),
		).
		Response()

	if !response.Okay() {
		return nil, response.Error()
	}

	job := &core.Job{}

	err := json.Unmarshal(response.Data(), job)
	if err != nil {
		return nil, err
	}

	return job, nil
}

func executeJob(driver *hype.Driver, jobGUID string) (*execution, error) {
	response := driver.
		Post("jobs/"+jobGUID+"/execute", nil, nil).
		WithHeaderSet(
			hype.NewHeader("Authorization", "jeremy"),
			hype.NewHeader("Accept", "application/json"),
			hype.NewHeader("Content-Type", "application/json"),
		).
		Response()

	if !response.Okay() {
		return nil, response.Error()
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
