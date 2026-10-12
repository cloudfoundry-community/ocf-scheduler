package broker

import (
	"fmt"

	"github.com/fivetwenty-io/osbapi/v2/pkg/osbapi"
)

// Cloud Controller tracks the offering by these IDs, so they never change.
// Changing either one orphans every instance and binding already created.
const (
	ServiceID   = "8fb69265-17c4-46fe-aa9e-80164e13865a"
	ServiceName = "scheduler"
	PlanID      = "1e83d77c-c04a-4190-a49c-dfeac779db7e"
	PlanName    = "standard"
)

// Catalog is the broker's one service and one free plan. Instances and
// bindings are retrievable, the plan cannot be changed, and the service
// carries no shareable metadata, so Cloud Controller refuses to share it.
func Catalog() *osbapi.Catalog {
	yes, no := true, false

	return &osbapi.Catalog{
		Services: []osbapi.Service{
			{
				ID:                   ServiceID,
				Name:                 ServiceName,
				Description:          "Run Cloud Foundry tasks and HTTP calls for an app on a cron schedule",
				Tags:                 []string{"scheduler", "cron"},
				Bindable:             true,
				InstancesRetrievable: true,
				BindingsRetrievable:  true,
				PlanUpdateable:       &no,
				Metadata: map[string]any{
					"displayName":      "Scheduler",
					"longDescription":  "A service instance turns the scheduler on for its space, and binding an app lets that app's jobs and calls be created, scheduled, and run.",
					"documentationUrl": "https://github.com/cloudfoundry-community/ocf-scheduler",
				},
				Plans: []osbapi.Plan{
					{
						ID:          PlanID,
						Name:        PlanName,
						Description: "Schedule jobs and calls for the apps bound in this space",
						Free:        &yes,
						Bindable:    &yes,
					},
				},
			},
		},
	}
}

// checkOffering refuses a request for any service or plan but ours.
func checkOffering(serviceID, planID string) error {
	if serviceID != ServiceID {
		return fmt.Errorf("%w: unknown service_id %q", osbapi.ErrBadRequest, serviceID)
	}

	if planID != PlanID {
		return fmt.Errorf("%w: unknown plan_id %q", osbapi.ErrBadRequest, planID)
	}

	return nil
}
