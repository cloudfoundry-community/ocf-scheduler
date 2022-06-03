package ops

import (
	"testing"

	"github.com/ess/testscope"

	"github.com/starkandwayne/scheduler-for-ocf/core"
	"github.com/starkandwayne/scheduler-for-ocf/core/failures"
	"github.com/starkandwayne/scheduler-for-ocf/mock"
)

func Test_LoadJob(t *testing.T) {
	testscope.SkipUnlessUnit(t)
	logScope := "ops.load-job"
	jobService := mock.NewJobService()
	logService := mock.NewLogService()
	services := &core.Services{Jobs: jobService, Logger: logService}
	guid, _ := core.GenGUID()
	job := dummyJob(&core.Job{GUID: guid, Name: "delete-job-test"})

	t.Run("when a guid is provided", func(t *testing.T) {
		input := core.
			NewInput(services).
			WithGUID(guid)

		t.Run("and the job exists", func(t *testing.T) {
			job, err := jobService.Persist(job)
			if err != nil {
				t.Errorf("could not persist job: %s", err.Error())
			}

			input = input.WithGUID(job.GUID)

			result := LoadJob(input)

			t.Run("the op is a success", func(t *testing.T) {
				if result.Failure() {
					cause := core.Causify(result.Error())
					t.Errorf("expected a success, got failure: %s, %s", cause, logService.ErrorsFor(logScope))
				}
			})

			t.Run("the payload has an executable job", func(t *testing.T) {})
		})

		t.Run("but the job doesn't exist", func(t *testing.T) {
			logService.Reset()
			jobService.Delete(job)

			result := LoadJob(input)

			t.Run("the op logs the issue with finding the job", func(t *testing.T) {
				if !logService.ReceivedError(logScope, "could not find job with guid "+job.GUID) {
					t.Errorf("expected a logged error regarding the unknown job %s, '%s'", job.GUID, logService.ErrorsFor(logScope))
				}
			})

			t.Run("the op is a failure", func(t *testing.T) {
				if result.Success() {
					t.Errorf("expected a failure, got success")
				}

				cause := core.Causify(result.Error())
				if cause != failures.NoSuchJob {
					t.Errorf("expected '%s', got '%s'", failures.NoSuchJob, cause)
				}
			})
		})
	})

	t.Run("when no guid is provided", func(t *testing.T) {
		logService.Reset()
		input := core.
			NewInput(services)

		result := LoadJob(input)

		t.Run("the op logs an error", func(t *testing.T) {
			if !logService.ReceivedError(logScope, "no job guid provided") {
				t.Errorf("expected a logged error regarding the missing guid")
			}
		})

		t.Run("the op is a failure", func(t *testing.T) {
			if result.Success() {
				t.Errorf("expected a failure, got success")
			}

			cause := core.Causify(result.Error())
			if cause != failures.NoSuchJob {
				t.Errorf("expected '%s', got '%s'", failures.NoSuchJob, cause)
			}
		})
	})

}
