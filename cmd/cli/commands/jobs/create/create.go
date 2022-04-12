package create

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
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

		params := hype.Params{}
		params.Set("app_guid", appGUID)

		payload := &scheduler.Job{
			Name:    jobName,
			Command: jobCommand,
		}

		data, err := json.Marshal(payload)

		response := core.Client.Post("jobs", params, data)

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
