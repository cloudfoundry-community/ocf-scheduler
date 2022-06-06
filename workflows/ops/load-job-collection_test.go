package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadJobCollection(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	jobService := mock.NewJobService()
	logService := mock.NewLogService()
	services := &core.Services{Jobs: jobService, Logger: logService}
	spaceGUID, _ := core.GenGUID()

	t.Run("when no space guid is provided", func(t *testing.T) {
		jobService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		result := LoadJobCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting job collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.JobCollection)

			if count != 0 {
				t.Errorf("expected no jobs, got %d", count)
			}
		})
	})

	t.Run("when the space has jobs", func(t *testing.T) {
		jobService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		for i := 0; i < 20; i++ {
			seed := &core.Job{SpaceGUID: spaceGUID}

			if _, err := jobService.Persist(dummyJob(seed)); err != nil {
				t.Errorf("could not persist job collection")
			}
		}

		result := LoadJobCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting job collection contains the proper jobs", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.JobCollection)

			if count != 20 {
				t.Errorf("expected 20 jobs, got %d", count)
			}
		})
	})

	t.Run("when the space has no jobs", func(t *testing.T) {
		jobService.Reset()
		input := core.
			NewInput(services).
			WithSpaceGUID(spaceGUID)

		result := LoadJobCollection(input)

		t.Run("the op is a success", func(t *testing.T) {
			if result.Failure() {
				cause := core.Causify(result.Error())

				t.Errorf("expected a success, got %s", cause)
			}
		})

		t.Run("the resulting job collection is empty", func(t *testing.T) {
			payload := core.Inputify(result.Value())
			count := len(payload.JobCollection)

			if count != 0 {
				t.Errorf("expected no job, got %d", count)
			}
		})
	})
}
