package del

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

type result struct {
	GUID    string `json:"guid"`
	Type    string `json:"type"`
	Result  string `json:"result"`
	Message string `json:"message"`
}

var Command = &cobra.Command{
	Use:   "del <job, call, or schedule GUID>...",
	Short: "Delete jobs and calls whose app is gone, and schedules whose job or call is gone",
	Long: `Delete jobs and calls whose app is gone, and schedules whose job or call is gone

Deletes each named job or call, with its schedules, only when Cloud
Controller reports its app as not found. It refuses any job or call whose
app exists or could not be checked.

A schedule GUID from the list of orphan schedules is deleted only when its
job or call is still gone, and it is refused with parent_exists otherwise.

The command fails if any GUID was not deleted.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cmd.Help()
			return fmt.Errorf("\nRequires at least one argument: <job, call, or schedule GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := json.Marshal(map[string][]string{"guids": args})
		if err != nil {
			return err
		}

		response, err := core.Client.Fetch(http.MethodPost, "orphans/delete", body)
		if err != nil {
			return err
		}

		if !response.Okay() {
			return response.AdminError()
		}

		data := struct {
			Results []*result `json:"results"`
		}{}

		err = json.Unmarshal(response.Body, &data)
		if err != nil {
			return err
		}

		writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "GUID\tType\tResult\tMessage")
		fmt.Fprintln(writer, "====\t====\t======\t=======")

		kept := 0
		for _, item := range data.Results {
			if item.Result != "deleted" {
				kept++
			}

			fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", item.GUID, item.Type, item.Result, item.Message)
		}

		writer.Flush()

		if kept > 0 {
			return fmt.Errorf("%d of %d were not deleted", kept, len(data.Results))
		}

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}
