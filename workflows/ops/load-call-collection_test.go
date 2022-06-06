package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadCallCollection(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	callService := mock.NewCallService()
	logService := mock.NewLogService()
	services := &core.Services{Calls: callService, Logger: logService}
	spaceGUID, _ := core.GenGUID()

	t.Run("when no space guid is provided", func(t *testing.T) {
		callService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		result := LoadCallCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting call collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.CallCollection)

			if count != 0 {
				t.Errorf("expected no calls, got %d", count)
			}
		})
	})

	t.Run("when the space has calls", func(t *testing.T) {
		callService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		for i := 0; i < 20; i++ {
			seed := &core.Call{SpaceGUID: spaceGUID}

			if _, err := callService.Persist(dummyCall(seed)); err != nil {
				t.Errorf("could not persist call collection")
			}
		}

		result := LoadCallCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting call collection contains the proper call", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.CallCollection)

			if count != 20 {
				t.Errorf("expected 20 calls, got %d", count)
			}
		})
	})

	t.Run("when the space has no calls", func(t *testing.T) {
		callService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		result := LoadCallCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting call collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.CallCollection)

			if count != 0 {
				t.Errorf("expected no calls, got %d", count)
			}
		})
	})
}
