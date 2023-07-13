package create

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "create <app GUID> <call name> <url> <auth header>",
	Short: "Create a call for an app",
	Long:  `Create a call for an app`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 4 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <space GUID> <call name> <URL> <auth header")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		appGUID := args[0]
		callName := args[1]
		callURL := args[2]
		callAuth := args[3]

		params := hype.Params{}
		params.Set("app_guid", appGUID)

		payload := &scheduler.Call{
			Name:       callName,
			URL:        callURL,
			AuthHeader: callAuth,
		}

		data, err := json.Marshal(payload)

		response := core.Client.Post("calls", params, data)

		if !response.Okay() {
			return response.Error()
		}

		err = json.Unmarshal(response.Data(), payload)
		if err != nil {
			return err
		}

		fmt.Printf(
			"Created call %s\n\tGUID: %s\n\tApp GUID: %s\n\tSpace GUID: %s\n\tURL: %s\n\tAuth Header: %s\n",
			payload.Name,
			payload.GUID,
			payload.AppGUID,
			payload.SpaceGUID,
			payload.URL,
			payload.AuthHeader,
		)

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}
