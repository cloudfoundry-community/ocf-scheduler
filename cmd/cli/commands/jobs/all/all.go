package all

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "all <space GUID>",
	Short: "List all jobs in a space",
	Long:  `List all jobs in a space`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <space GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		params := hype.Params{}
		params.Set("space_guid", args[0])

		response := core.Client.Get("jobs", params)

		if !response.Okay() {
			return response.Error()
		}

		data := struct {
			Resources []*scheduler.Job `json:"resources"`
		}{}

		err := json.Unmarshal(response.Data(), &data)
		if err != nil {
			return err
		}

		for _, job := range data.Resources {
			fmt.Printf("%s (%s)\n", job.Name, job.GUID)
		}

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}
