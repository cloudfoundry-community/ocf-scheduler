package all

import (
	"encoding/json"
	"fmt"

	"github.com/ess/hype"
	"github.com/spf13/cobra"
	"github.com/starkandwayne/scheduler-for-ocf/core"
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
		driver, err := hype.New("http://localhost:8000")
		if err != nil {
			return fmt.Errorf("couldn't hype it up: %s", err.Error())
		}

		params := hype.Params{}
		params.Set("space_guid", args[0])

		response := driver.
			Get("jobs", params).
			WithHeaderSet(
				hype.NewHeader("Authorization", "jeremy"),
				hype.NewHeader("Accept", "application/json"),
				hype.NewHeader("Content-Type", "application/json"),
			).
			Response()

		if !response.Okay() {
			return response.Error()
		}

		data := struct {
			Resources []*core.Job `json:"resources"`
		}{}

		err = json.Unmarshal(response.Data(), &data)
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
