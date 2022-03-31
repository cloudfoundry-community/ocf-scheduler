package create

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/scheduler-for-ocf/core"
)

var Command = &cobra.Command{
	Use:   "create <app GUID> <job name> <command>",
	Short: "Create a job for an app",
	Long:  `Create a job for an app`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 3 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <space GUID> <job name> <command>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		appGUID := args[0]
		jobName := args[1]
		jobCommand := args[2]

		driver, err := hype.New("http://localhost:8000")
		if err != nil {
			return fmt.Errorf("couldn't hype it up: %s", err.Error())
		}

		params := hype.Params{}
		params.Set("app_guid", appGUID)

		payload := &core.Job{
			Name:    jobName,
			Command: jobCommand,
		}

		data, err := json.Marshal(payload)

		response := driver.
			Post("jobs", params, data).
			WithHeaderSet(
				hype.NewHeader("Authorization", "jeremy"),
				hype.NewHeader("Accept", "application/json"),
				hype.NewHeader("Content-Type", "application/json"),
			).
			Response()

		if !response.Okay() {
			return response.Error()
		}

		err = json.Unmarshal(response.Data(), payload)
		if err != nil {
			return err
		}

		fmt.Printf(
			"Created job %s\n\tGUID: %s\n\tApp GUID: %s\n\tSpace GUID: %s\n\tCommand: %s\n",
			payload.Name,
			payload.GUID,
			payload.AppGUID,
			payload.SpaceGUID,
			payload.Command,
		)

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}
