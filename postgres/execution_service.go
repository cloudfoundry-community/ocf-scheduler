package postgres

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/starkandwayne/scheduler-for-ocf/core"
)

type ExecutionService struct {
	db *sql.DB
}

func NewExecutionService(db *sql.DB) *ExecutionService {
	return &ExecutionService{db}
}

func (service *ExecutionService) Get(guid string) (*core.Execution, error) {
	candidates := service.getCollection(
		"select * from executions where guid = $1",
		guid,
	)

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

func (service *ExecutionService) ByCall(call *core.Call) []*core.Execution {
	return service.getCollection(
		"select * from executions where ref_guid = $1 and ref_type = $2",
		call.GUID,
		"call",
	)
}

func (service *ExecutionService) ByJob(job *core.Job) []*core.Execution {
	return service.getCollection(
		"select * from executions where ref_type = $1 and ref_guid = $2",
		"job",
		job.GUID,
	)
}

func (service *ExecutionService) BySchedule(schedule *core.Schedule) []*core.Execution {
	return service.getCollection(
		"select * from executions where schedule_guid = $1",
		schedule.GUID,
	)
}

func (service *ExecutionService) Persist(candidate *core.Execution) (*core.Execution, error) {
	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("could not generate an execution id")
	}

	candidate.GUID = guid

	return service.insert(candidate)
}

func (service *ExecutionService) Fail(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "FAILED")
}

func (service *ExecutionService) Success(execution *core.Execution) (*core.Execution, error) {
	return service.finish(execution, "SUCCEEDED")
}

func (service *ExecutionService) UpdateMessage(execution *core.Execution, message string) (*core.Execution, error) {
	execution.Message = message

	return service.update(execution)
}

func (service *ExecutionService) Start(execution *core.Execution) (*core.Execution, error) {
	now := time.Now().UTC()

	execution.ExecutionStartTime = now

	if len(execution.ScheduleGUID) > 0 {
		execution.ScheduledTime = time.Date(
			now.Year(),
			now.Month(),
			now.Day(),
			now.Hour(),
			now.Minute(),
			0,
			0,
			time.UTC,
		)
	}

	return service.update(execution)
}

func (service *ExecutionService) getCollection(query string, args ...interface{}) []*core.Execution {
	collection := make([]*core.Execution, 0)

	rows, err := service.db.Query(query, args...)
	if err != nil {
		return collection
	}

	for rows.Next() {
		var guid string
		var refGUID string
		var refType string
		var taskGUID string
		var scheduleGUID string
		var scheduledTime time.Time
		var message string
		var state string
		var executionStartTime time.Time
		var executionEndTime time.Time

		err := rows.Scan(&guid, &refGUID, &refType, &taskGUID, &scheduleGUID, &scheduledTime, &message, &state, &executionStartTime, &executionEndTime)
		if err != nil {
			continue
		}

		candidate := &core.Execution{
			GUID:               guid,
			RefGUID:            refGUID,
			RefType:            refType,
			TaskGUID:           taskGUID,
			ScheduleGUID:       scheduleGUID,
			ScheduledTime:      scheduledTime,
			Message:            message,
			State:              state,
			ExecutionStartTime: executionStartTime,
			ExecutionEndTime:   executionEndTime,
		}

		collection = append(collection, candidate)
	}

	return collection
}

func (service *ExecutionService) insert(candidate *core.Execution) (*core.Execution, error) {
	err := WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"insert into executions values($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)",
			candidate.GUID,
			candidate.RefGUID,
			candidate.RefType,
			candidate.TaskGUID,
			candidate.ScheduleGUID,
			candidate.ScheduledTime,
			candidate.Message,
			candidate.State,
			candidate.ExecutionStartTime,
			candidate.ExecutionEndTime,
		)

		return aErr
	})

	if err != nil {
		return nil, err
	}

	return candidate, nil
}

func (service *ExecutionService) update(candidate *core.Execution) (*core.Execution, error) {
	err := WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			"update executions set ref_guid = $2, ref_type = $3, task_guid = $4, schedule_guid = $5, scheduled_time = $6, message = $7, state = $8, execution_start_time = $9, execution_end_time = $10 where guid = $1",
			candidate.GUID,
			candidate.RefGUID,
			candidate.RefType,
			candidate.TaskGUID,
			candidate.ScheduleGUID,
			candidate.ScheduledTime,
			candidate.Message,
			candidate.State,
			candidate.ExecutionStartTime,
			candidate.ExecutionEndTime,
		)

		return aErr
	})

	if err != nil {
		return nil, err
	}

	return candidate, nil
}

func (service *ExecutionService) finish(execution *core.Execution, state string) (*core.Execution, error) {
	execution.ExecutionEndTime = time.Now().UTC()
	execution.State = state

	return service.update(execution)
}
