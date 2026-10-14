package history

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/spf13/cobra"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "history <job GUID>",
	Short: "Get a job's execution history",
	Long:  `Get a job's execution history`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <job GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		jobGUID := args[0]

		_, err := getJob(core.Client, jobGUID)
		if err != nil {
			return err
		}

		executions, err := getJobExecutions(core.Client, jobGUID)
		if err != nil {
			return err
		}

		if len(executions) == 0 {
			fmt.Println("No executions for job", jobGUID)

			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 0, 3, ' ', uint(0))
		fmt.Fprintln(writer, "State\tStart Time\tEnd Time")
		fmt.Fprintln(writer, "=====\t====================\t==================")

		for _, execution := range executions {
			fmt.Fprintf(
				writer,
				"%s\t%s\t%s\n",
				execution.State,
				execution.ExecutionStartTime,
				execution.ExecutionEndTime,
			)
		}

		writer.Flush()

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

func getJobExecutions(driver *core.Driver, jobGUID string) ([]*execution, error) {
	response := driver.Get("jobs/"+jobGUID+"/history", nil)

	if !response.Okay() {
		return nil, response.Error()
	}

	data := struct {
		Resources []*execution `json:"resources"`
	}{}

	err := json.Unmarshal(response.Data(), &data)
	if err != nil {
		return nil, err
	}

	return data.Resources, nil
}

type execution struct {
	State              string    `json:"state"`
	ExecutionStartTime time.Time `json:"execution_start_time"`
	ExecutionEndTime   time.Time `json:"execution_end_time"`
}
