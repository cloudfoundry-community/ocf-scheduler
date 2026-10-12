// Package broker serves the Open Service Broker API for the scheduler. A
// service instance turns the scheduler on for one space, and a binding lets
// one app in that space be scheduled. Binding credentials carry the scheduler
// URL and no secret, because the scheduler API keeps authenticating each
// caller with their own Cloud Foundry token.
package broker

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"

	osbbroker "github.com/fivetwenty-io/osbapi/v2/pkg/broker"
	"github.com/fivetwenty-io/osbapi/v2/pkg/osbapi"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/workflows"
)

// Broker implements osbapi.ServiceBroker on the scheduler's stores. Every
// operation finishes before it answers, so none is ever asynchronous.
type Broker struct {
	services *core.Services
	apiURL   string

	// locker serializes the operations that change instances, bindings, and
	// schedules, so an unbind and a bind for the same app cannot interleave.
	locker sync.Mutex
}

var _ osbapi.ServiceBroker = (*Broker)(nil)

// New builds a broker whose binding credentials point at apiURL.
func New(services *core.Services, apiURL string) *Broker {
	return &Broker{services: services, apiURL: apiURL}
}

// NewHandler serves the broker under /v2 behind HTTP basic auth with the
// configured credentials. It refuses to build a handler without both
// credentials, because the osbapi handler skips auth when both are empty.
func NewHandler(services *core.Services, config Config) (http.Handler, error) {
	if len(config.Username) == 0 || len(config.Password) == 0 {
		return nil, fmt.Errorf("the service broker needs both a username and a password")
	}

	if services.Instances == nil || services.Bindings == nil {
		return nil, fmt.Errorf("the service broker needs the instance and binding stores")
	}

	return osbbroker.NewHandler(
		New(services, config.APIURL),
		osbbroker.WithBasicAuth(config.Username, config.Password),
		osbbroker.WithLogger(logAdapter{log: services.Logger}),
	), nil
}

func (b *Broker) GetCatalog(_ context.Context) (*osbapi.Catalog, error) {
	return Catalog(), nil
}

func (b *Broker) Provision(ctx context.Context, instanceID string, req osbapi.ProvisionRequest, _ bool) (osbapi.ProvisionResponse, bool, error) {
	tag := "broker-provision"
	none := osbapi.ProvisionResponse{}

	if err := checkOffering(req.ServiceID, req.PlanID); err != nil {
		return none, false, err
	}

	spaceGUID := firstOf(req.SpaceGUID, contextString(req.Context, "space_guid"))
	orgGUID := firstOf(req.OrganizationGUID, contextString(req.Context, "organization_guid"))
	if len(spaceGUID) == 0 {
		return none, false, fmt.Errorf("%w: space_guid is required", osbapi.ErrBadRequest)
	}

	b.locker.Lock()
	defer b.locker.Unlock()

	existing, err := b.services.Instances.Get(instanceID)
	switch {
	case err == nil:
		if existing.ServiceID == req.ServiceID && existing.PlanID == req.PlanID &&
			existing.SpaceGUID == spaceGUID && existing.OrgGUID == orgGUID {
			return none, false, osbapi.ErrInstanceAlreadyProvisioned
		}
		return none, false, fmt.Errorf("%w: instance %s exists with different attributes", osbapi.ErrInstanceAlreadyExists, instanceID)
	case !errors.Is(err, core.ErrNotFound):
		return none, false, b.storeError(tag, "look up service instance "+instanceID, err)
	}

	if err := b.clearSpace(tag, spaceGUID); err != nil {
		return none, false, err
	}

	_, err = b.services.Instances.Persist(&core.ServiceInstance{
		InstanceID: instanceID,
		ServiceID:  req.ServiceID,
		PlanID:     req.PlanID,
		OrgGUID:    orgGUID,
		SpaceGUID:  spaceGUID,
		Context:    req.Context,
	})
	if errors.Is(err, core.ErrConflict) {
		return none, false, spaceTaken()
	}
	if err != nil {
		return none, false, b.storeError(tag, "store service instance "+instanceID, err)
	}

	b.services.Logger.Info(tag, fmt.Sprintf("provisioned instance %s in space %s%s", instanceID, spaceGUID, requester(ctx)))

	return none, false, nil
}

// clearSpace makes room for a new instance in the space. Another instance
// there blocks the new one unless Cloud Controller no longer knows it, which
// happens after cf purge-service-instance skips the broker. That orphan is
// removed the way a deprovision would remove it.
func (b *Broker) clearSpace(tag, spaceGUID string) error {
	other, err := b.services.Instances.InSpace(spaceGUID)
	if errors.Is(err, core.ErrNotFound) {
		return nil
	}
	if err != nil {
		return b.storeError(tag, "look up the instance in space "+spaceGUID, err)
	}

	exists, err := b.services.Info.ServiceInstanceExists(other.InstanceID)
	if err != nil {
		b.services.Logger.Error(tag, fmt.Sprintf("could not ask Cloud Controller about instance %s: %v", other.InstanceID, err))
		return fmt.Errorf("space %s already has scheduler instance %s, and Cloud Controller could not be asked whether it still exists; try again", spaceGUID, other.InstanceID)
	}

	if exists {
		return spaceTaken()
	}

	b.services.Logger.Warn(tag, fmt.Sprintf("replacing instance %s in space %s, which Cloud Controller no longer knows", other.InstanceID, spaceGUID))

	return b.removeInstance(tag, other.InstanceID)
}

func (b *Broker) Deprovision(ctx context.Context, instanceID string, req osbapi.DeprovisionRequest, _ bool) (osbapi.DeprovisionResponse, bool, error) {
	tag := "broker-deprovision"
	none := osbapi.DeprovisionResponse{}

	if err := checkGivenOffering(req.ServiceID, req.PlanID); err != nil {
		return none, false, err
	}

	b.locker.Lock()
	defer b.locker.Unlock()

	if _, err := b.services.Instances.Get(instanceID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return none, false, fmt.Errorf("%w: instance %s does not exist", osbapi.ErrGoneError, instanceID)
		}
		return none, false, b.storeError(tag, "look up service instance "+instanceID, err)
	}

	if err := b.removeInstance(tag, instanceID); err != nil {
		return none, false, err
	}

	b.services.Logger.Info(tag, fmt.Sprintf("deprovisioned instance %s%s", instanceID, requester(ctx)))

	return none, false, nil
}

// removeInstance deletes any bindings left under the instance, pausing each
// app that loses its last one, and then deletes the instance. Jobs, calls,
// schedules, and history all stay.
func (b *Broker) removeInstance(tag, instanceID string) error {
	bindings, err := b.services.Bindings.ForInstance(instanceID)
	if err != nil {
		return b.storeError(tag, "list the bindings of instance "+instanceID, err)
	}

	for _, binding := range bindings {
		if err := b.removeBinding(tag, binding); err != nil {
			return err
		}
	}

	if err := b.services.Instances.Delete(instanceID); err != nil {
		return b.storeError(tag, "delete service instance "+instanceID, err)
	}

	return nil
}

func (b *Broker) GetInstance(_ context.Context, instanceID string, _ osbapi.FetchInstanceRequest) (osbapi.FetchInstanceResponse, error) {
	instance, err := b.services.Instances.Get(instanceID)
	if errors.Is(err, core.ErrNotFound) {
		return osbapi.FetchInstanceResponse{}, osbapi.ErrInstanceNotFound
	}
	if err != nil {
		return osbapi.FetchInstanceResponse{}, b.storeError("broker-get-instance", "look up service instance "+instanceID, err)
	}

	return osbapi.FetchInstanceResponse{
		ServiceID: instance.ServiceID,
		PlanID:    instance.PlanID,
	}, nil
}

// Update has nothing to change. It answers 200 for the instance's own plan
// and refuses any other, since the plan is not updateable.
func (b *Broker) Update(_ context.Context, instanceID string, req osbapi.UpdateRequest, _ bool) (osbapi.UpdateResponse, bool, error) {
	if req.ServiceID != ServiceID {
		return osbapi.UpdateResponse{}, false, fmt.Errorf("%w: unknown service_id %q", osbapi.ErrBadRequest, req.ServiceID)
	}

	if len(req.PlanID) > 0 && req.PlanID != PlanID {
		return osbapi.UpdateResponse{}, false, fmt.Errorf("%w: the plan cannot be changed", osbapi.ErrBadRequest)
	}

	if _, err := b.services.Instances.Get(instanceID); err != nil {
		if errors.Is(err, core.ErrNotFound) {
			return osbapi.UpdateResponse{}, false, osbapi.ErrInstanceNotFound
		}
		return osbapi.UpdateResponse{}, false, b.storeError("broker-update", "look up service instance "+instanceID, err)
	}

	return osbapi.UpdateResponse{}, false, nil
}

func (b *Broker) LastOperation(_ context.Context, _ string, _ osbapi.LastOperationRequest) (osbapi.LastOperationResponse, error) {
	return osbapi.LastOperationResponse{}, noOperations()
}

func (b *Broker) Bind(ctx context.Context, instanceID, bindingID string, req osbapi.BindRequest, _ bool) (osbapi.BindResponse, bool, error) {
	tag := "broker-bind"
	none := osbapi.BindResponse{}

	if err := checkOffering(req.ServiceID, req.PlanID); err != nil {
		return none, false, err
	}

	appGUID := req.AppGUID
	if len(appGUID) == 0 && req.BindResource != nil {
		appGUID = req.BindResource.AppGUID
	}

	b.locker.Lock()
	defer b.locker.Unlock()

	instance, err := b.services.Instances.Get(instanceID)
	if errors.Is(err, core.ErrNotFound) {
		return none, false, osbapi.ErrInstanceNotFound
	}
	if err != nil {
		return none, false, b.storeError(tag, "look up service instance "+instanceID, err)
	}

	existing, err := b.services.Bindings.Get(bindingID)
	switch {
	case err == nil:
		if existing.InstanceID != instanceID || existing.AppGUID != appGUID {
			return none, false, fmt.Errorf("%w: binding %s exists with different attributes", osbapi.ErrBindingAlreadyExists, bindingID)
		}

		// Resume again in case the first attempt stored the binding and then
		// failed, so a repeated bind always leaves the app running.
		if err := b.resume(tag, appGUID); err != nil {
			return none, false, err
		}

		return osbapi.BindResponse{Credentials: b.credentials(instanceID, appGUID)}, false, osbapi.ErrBindingAlreadyExists
	case !errors.Is(err, core.ErrNotFound):
		return none, false, b.storeError(tag, "look up service binding "+bindingID, err)
	}

	if len(appGUID) > 0 {
		spaceGUID, err := b.services.Info.GetSpaceGUIDForApp(appGUID)
		if err != nil {
			b.services.Logger.Error(tag, fmt.Sprintf("could not find the space of app %s: %v", appGUID, err))
			return none, false, fmt.Errorf("could not find the space of app %s in Cloud Controller; try again", appGUID)
		}

		if spaceGUID != instance.SpaceGUID {
			return none, false, fmt.Errorf("%w: app %s is not in the space of instance %s", osbapi.ErrBadRequest, appGUID, instanceID)
		}
	}

	_, err = b.services.Bindings.Persist(&core.ServiceBinding{
		BindingID:  bindingID,
		InstanceID: instanceID,
		AppGUID:    appGUID,
	})
	if errors.Is(err, core.ErrConflict) {
		return none, false, fmt.Errorf("%w: binding %s already exists", osbapi.ErrBindingAlreadyExists, bindingID)
	}
	if err != nil {
		return none, false, b.storeError(tag, "store service binding "+bindingID, err)
	}

	if err := b.resume(tag, appGUID); err != nil {
		return none, false, err
	}

	if len(appGUID) > 0 {
		b.services.Logger.Info(tag, fmt.Sprintf("bound app %s to instance %s as binding %s%s", appGUID, instanceID, bindingID, requester(ctx)))
	} else {
		b.services.Logger.Info(tag, fmt.Sprintf("created service key %s on instance %s%s", bindingID, instanceID, requester(ctx)))
	}

	return osbapi.BindResponse{Credentials: b.credentials(instanceID, appGUID)}, false, nil
}

func (b *Broker) Unbind(ctx context.Context, instanceID, bindingID string, req osbapi.UnbindRequest, _ bool) (osbapi.UnbindResponse, bool, error) {
	tag := "broker-unbind"
	none := osbapi.UnbindResponse{}

	if err := checkGivenOffering(req.ServiceID, req.PlanID); err != nil {
		return none, false, err
	}

	b.locker.Lock()
	defer b.locker.Unlock()

	binding, err := b.services.Bindings.Get(bindingID)
	if errors.Is(err, core.ErrNotFound) || (err == nil && binding.InstanceID != instanceID) {
		return none, false, fmt.Errorf("%w: binding %s does not exist", osbapi.ErrGoneError, bindingID)
	}
	if err != nil {
		return none, false, b.storeError(tag, "look up service binding "+bindingID, err)
	}

	if err := b.removeBinding(tag, binding); err != nil {
		return none, false, err
	}

	b.services.Logger.Info(tag, fmt.Sprintf("removed binding %s from instance %s%s", bindingID, instanceID, requester(ctx)))

	return none, false, nil
}

// removeBinding deletes a binding and pauses its app's schedules when the
// app has no other binding. It pauses first, so a failure leaves the binding
// in place for Cloud Controller to retry, and it resumes again if another
// binding still names the app.
func (b *Broker) removeBinding(tag string, binding *core.ServiceBinding) error {
	if len(binding.AppGUID) > 0 {
		if _, err := workflows.PausingAnApp(b.services, binding.AppGUID); err != nil {
			return err
		}
	}

	if err := b.services.Bindings.Delete(binding.BindingID); err != nil {
		return b.storeError(tag, "delete service binding "+binding.BindingID, err)
	}

	if len(binding.AppGUID) == 0 {
		return nil
	}

	bound, err := b.services.Bindings.Bound(binding.AppGUID)
	if err != nil {
		return b.storeError(tag, "check the other bindings of app "+binding.AppGUID, err)
	}

	if bound {
		return b.resume(tag, binding.AppGUID)
	}

	return nil
}

func (b *Broker) GetBinding(_ context.Context, instanceID, bindingID string, _ osbapi.FetchBindingRequest) (osbapi.FetchBindingResponse, error) {
	binding, err := b.services.Bindings.Get(bindingID)
	if errors.Is(err, core.ErrNotFound) || (err == nil && binding.InstanceID != instanceID) {
		return osbapi.FetchBindingResponse{}, osbapi.ErrBindingNotFound
	}
	if err != nil {
		return osbapi.FetchBindingResponse{}, b.storeError("broker-get-binding", "look up service binding "+bindingID, err)
	}

	return osbapi.FetchBindingResponse{Credentials: b.credentials(instanceID, binding.AppGUID)}, nil
}

func (b *Broker) LastBindingOperation(_ context.Context, _, _ string, _ osbapi.LastOperationRequest) (osbapi.LastOperationResponse, error) {
	return osbapi.LastOperationResponse{}, noOperations()
}

// credentials hold the scheduler URL and IDs and never a secret.
func (b *Broker) credentials(instanceID, appGUID string) map[string]any {
	credentials := map[string]any{
		"api_url":     b.apiURL,
		"instance_id": instanceID,
	}

	if len(appGUID) > 0 {
		credentials["app_guid"] = appGUID
	}

	return credentials
}

// resume re-enables the schedules that unbinding paused. A service key has
// no app and resumes nothing.
func (b *Broker) resume(tag, appGUID string) error {
	if len(appGUID) == 0 {
		return nil
	}

	if _, err := workflows.ResumingAnApp(b.services, appGUID); err != nil {
		b.services.Logger.Error(tag, err.Error())
		return fmt.Errorf("could not resume the schedules of app %s; try again", appGUID)
	}

	return nil
}

// storeError logs a store failure in full and answers Cloud Controller with
// a description that names the step but not the database's message.
func (b *Broker) storeError(tag, step string, err error) error {
	b.services.Logger.Error(tag, fmt.Sprintf("could not %s: %v", step, err))
	return fmt.Errorf("could not %s; try again", step)
}

func spaceTaken() error {
	return &osbapi.OSBError{
		StatusCode:  http.StatusConflict,
		Description: "this space already has a scheduler service instance, and a space can hold only one",
	}
}

func noOperations() error {
	return &osbapi.OSBError{
		StatusCode:  http.StatusBadRequest,
		Description: "the scheduler broker finishes every operation before it answers, so there is no operation to poll",
	}
}

// checkGivenOffering checks the service and plan IDs a delete names, which
// Cloud Controller always sends but the API lets a client leave out.
func checkGivenOffering(serviceID, planID string) error {
	if len(serviceID) > 0 && serviceID != ServiceID {
		return fmt.Errorf("%w: unknown service_id %q", osbapi.ErrBadRequest, serviceID)
	}

	if len(planID) > 0 && planID != PlanID {
		return fmt.Errorf("%w: unknown plan_id %q", osbapi.ErrBadRequest, planID)
	}

	return nil
}

// requester names the Cloud Foundry user from the originating identity
// header for the audit log, or nothing when the header was absent.
func requester(ctx context.Context) string {
	identity, found := osbapi.OriginatingIdentityFromContext(ctx)
	if !found {
		return ""
	}

	if user, ok := identity.Value["user_id"].(string); ok && len(user) > 0 {
		return fmt.Sprintf(" for %s user %s", identity.Platform, user)
	}

	return ""
}

func contextString(context map[string]any, key string) string {
	if value, ok := context[key].(string); ok {
		return value
	}

	return ""
}

func firstOf(values ...string) string {
	for _, value := range values {
		if len(value) > 0 {
			return value
		}
	}

	return ""
}
