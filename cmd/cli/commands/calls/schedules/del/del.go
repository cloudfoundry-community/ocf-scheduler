package del

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	scheduler "github.com/cloudfoundry-community/ocf-scheduler/core"

	"github.com/cloudfoundry-community/ocf-scheduler/cmd/cli/core"
)

var Command = &cobra.Command{
	Use:   "del <call GUID> <schedule GUID>",
	Short: "Delete a schedule for a call",
	Long:  `Delete a schedule for a call`,
	Args: func(cmd *cobra.Command, args []string) error {
		if len(args) != 2 {
			cmd.Help()
			return fmt.Errorf("\nRequires one argument: <call GUID> <schedule GUID")
		}

		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		callGUID := args[0]
		scheduleGUID := args[1]

		call, err := getCall(core.Client, callGUID)
		if err != nil {
			return fmt.Errorf("couldn't find that call")
		}

		err = deleteSchedule(core.Client, call.GUID, scheduleGUID)
		if err != nil {
			return fmt.Errorf("couldn't delete schedule %s", scheduleGUID)
		}

		fmt.Println("Schedule", scheduleGUID, "deleted.")
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

func deleteSchedule(driver *core.Driver, callGUID string, scheduleGUID string) error {
	response := core.Client.Delete(
		"calls/"+callGUID+"/schedules/"+scheduleGUID,
		nil,
	)

	if !response.Okay() {
		return response.Error()
	}

	return nil
}
