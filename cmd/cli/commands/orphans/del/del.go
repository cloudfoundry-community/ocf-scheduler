package del

import (
	"encoding/json"
	"fmt"
	"os"
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
	Use:   "del <job or call GUID>...",
	Short: "Delete jobs and calls whose app is gone",
	Long: `Delete jobs and calls whose app is gone

Deletes each named job or call, with its schedules, only when Cloud
Controller reports its app as not found. It refuses any job or call whose
app exists or could not be checked, and it fails if any GUID was not
deleted.`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			cmd.Help()
			return fmt.Errorf("\nRequires at least one argument: <job or call GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		body, err := json.Marshal(map[string][]string{"guids": args})
		if err != nil {
			return err
		}

		response := core.Client.Post("orphans/delete", nil, body)

		if !response.Okay() {
			return response.Error()
		}

		data := struct {
			Results []*result `json:"results"`
		}{}

		err = json.Unmarshal(response.Data(), &data)
		if err != nil {
			return err
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
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
