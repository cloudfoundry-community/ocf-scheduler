package execute

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	scheduler "github.com/starkandwayne/scheduler-for-ocf/core"

	"github.com/starkandwayne/scheduler-for-ocf/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "execute <call GUID>",
	Short: "Execute a call",
	Long:  `Execute a call`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 1 {
			cmd.Help()
			return fmt.Errorf("\nRequired arguments: <call GUID>")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]

		call, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		execution, err := executeCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't execute that call")
		}

		fmt.Printf(
			"Executed call %s (%s) [Execution GUID: %s]\n",
			call.Name,
			call.GUID,
			execution.GUID,
		)

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

func getCall(driver *core.Driver, callGUID string) (*scheduler.Call, error) {
	response := driver.Get("calls/"+callGUID, nil)

	if !response.Okay() {
		return nil, response.Error()
	}

	call := &scheduler.Call{}

	err := json.Unmarshal(response.Data(), call)
	if err != nil {
		return nil, err
	}

	return call, nil
}

func executeCall(driver *core.Driver, callGUID string) (*execution, error) {
	response := driver.Post("calls/"+callGUID+"/execute", nil, nil)

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
