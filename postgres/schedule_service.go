package postgres

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/lib/pq"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

var scheduleColumns = []string{
	"guid",            // CHAR(36) PRIMARY KEY,
	"enabled",         // BOOL,
	"expression",      // TEXT NOT NULL,
	"expression_type", // TEXT NOT NULL,
	"ref_guid",        // CHAR(36) NOT NULL,
	"ref_type",        // TEXT NOT NULL,
	"created_at",      // TIMESTAMP WITH TIME ZONE NOT NULL,
	"updated_at",      // TIMESTAMP WITH TIME ZONE NOT NULL
	"paused_by",       // TEXT DEFAULT NULL
}

// appSchedules matches the schedules on the jobs and calls of the app in $1.
const appSchedules = `((ref_type = 'job' AND ref_guid IN (SELECT guid FROM jobs WHERE app_guid = $1)) OR
	(ref_type = 'call' AND ref_guid IN (SELECT guid FROM calls WHERE app_guid = $1)))`

// orphanedSchedules matches the schedules whose job or call is not stored.
// It goes by ref_type, as loading schedules at startup does, so a schedule
// that names the wrong kind of owner counts as orphaned too.
const orphanedSchedules = `NOT EXISTS (SELECT 1 FROM jobs WHERE schedules.ref_type = 'job' AND jobs.guid = schedules.ref_guid) AND
	NOT EXISTS (SELECT 1 FROM calls WHERE schedules.ref_type = 'call' AND calls.guid = schedules.ref_guid)`

type ScheduleService struct {
	db                  *sql.DB
	queryByPrimaryKey   string
	queryByEnabled      string
	queryByCall         string
	queryByJob          string
	queryByApp          string
	queryOrphaned       string
	pauseByApp          string
	resumeByApp         string
	insertIntoSchedules string
}

func NewScheduleService(db *sql.DB) *ScheduleService {
	qbpk := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE guid = $1"
	qbc := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE ref_type = 'call' AND ref_guid = $1"
	qbe := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE enabled"
	qbj := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE ref_type = 'job' AND ref_guid = $1"

	qba := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE " + appSchedules
	qo := "SELECT " + strings.Join(scheduleColumns, ", ") + " FROM schedules WHERE " + orphanedSchedules
	pba := "UPDATE schedules SET enabled = false, paused_by = $2, updated_at = $3 WHERE enabled AND " +
		appSchedules + " RETURNING " + strings.Join(scheduleColumns, ", ")
	rba := "UPDATE schedules SET enabled = true, paused_by = NULL, updated_at = $3 WHERE paused_by = $2 AND guid = ANY($4::text[]) AND " +
		appSchedules + " RETURNING " + strings.Join(scheduleColumns, ", ")

	is := generateInsert("schedules", scheduleColumns)
	return &ScheduleService{
		db:                  db,
		queryByPrimaryKey:   qbpk,
		queryByEnabled:      qbe,
		queryByCall:         qbc,
		queryByJob:          qbj,
		queryByApp:          qba,
		queryOrphaned:       qo,
		pauseByApp:          pba,
		resumeByApp:         rba,
		insertIntoSchedules: is,
	}
}

// Get reports a failed query, because a lost database must not read as a
// schedule that does not exist.
func (service *ScheduleService) Get(guid string) (*core.Schedule, error) {
	candidates, err := service.query(
		service.queryByPrimaryKey, // "select * from schedules where guid = $1"
		guid,
	)
	if err != nil {
		return nil, err
	}

	if err := expectingOne(len(candidates)); err != nil {
		return nil, err
	}

	return candidates[0], nil
}

// ByCall reports a failed query, because deleting a call must not mistake a
// lost database for a call without schedules.
func (service *ScheduleService) ByCall(call *core.Call) ([]*core.Schedule, error) {
	return service.query(
		service.queryByCall, // "select * from schedules where ref_type = 'call' and ref_guid = $1"
		call.GUID,
	)
}

func (service *ScheduleService) Enabled() []*core.Schedule {
	return service.getCollection(
		service.queryByEnabled, // "select * from schedules where enabled"
	)
}

// ByJob reports a failed query, because deleting a job must not mistake a
// lost database for a job without schedules.
func (service *ScheduleService) ByJob(job *core.Job) ([]*core.Schedule, error) {
	return service.query(
		service.queryByJob, // "select * from schedules where ref_type = 'job' and ref_guid = $1",
		job.GUID,
	)
}

// ByApp returns the schedules on the app's jobs and calls. Unlike the older
// lookups, it reports a failed query, because pausing an app must not
// mistake a lost database for an app without schedules.
func (service *ScheduleService) ByApp(appGUID string) ([]*core.Schedule, error) {
	rows, err := service.db.Query(service.queryByApp, appGUID)
	if err != nil {
		return nil, err
	}

	return scanSchedules(rows)
}

func (service *ScheduleService) Orphaned() ([]*core.Schedule, error) {
	return service.query(service.queryOrphaned)
}

func (service *ScheduleService) Pause(appGUID string, reason string) ([]*core.Schedule, error) {
	return service.flip(service.pauseByApp, appGUID, reason)
}

func (service *ScheduleService) Resume(appGUID string, reason string, guids []string) ([]*core.Schedule, error) {
	return service.flip(service.resumeByApp, appGUID, reason, pq.Array(guids))
}

// flip runs a pause or resume update, which changes every matching schedule
// in one statement, and returns the rows it changed. A resume passes the
// schedule GUIDs to change as extra.
func (service *ScheduleService) flip(query, appGUID, reason string, extra ...any) ([]*core.Schedule, error) {
	var changed []*core.Schedule

	args := append([]any{appGUID, reason, time.Now().UTC()}, extra...)

	err := WithTransaction(service.db, func(tx Transaction) error {
		rows, qErr := tx.Query(query, args...)
		if qErr != nil {
			return qErr
		}

		var sErr error
		changed, sErr = scanSchedules(rows)
		return sErr
	})

	if err != nil {
		return nil, err
	}

	return changed, nil
}

func (service *ScheduleService) Delete(schedule *core.Schedule) error {
	// Let's not try to delete something that isn't in the db. A lookup that
	// fails is not the same as a schedule that isn't there, so it is returned.
	found, lErr := service.query(service.queryByPrimaryKey, schedule.GUID)
	if lErr != nil {
		return lErr
	}

	if len(found) != 1 {
		return nil
	}

	err := WithTransaction(service.db, func(tx Transaction) error {
		_, dErr := tx.Exec(
			"DELETE FROM schedules WHERE guid = $1",
			schedule.GUID,
		)

		return dErr
	})

	return err
}

func (service *ScheduleService) Persist(candidate *core.Schedule) (*core.Schedule, error) {
	now := time.Now().UTC()

	guid, err := core.GenGUID()
	if err != nil {
		return nil, fmt.Errorf("coult not generate a schedule id")
	}

	candidate.GUID = guid
	candidate.CreatedAt = now
	candidate.UpdatedAt = now

	err = WithTransaction(service.db, func(tx Transaction) error {
		_, aErr := tx.Exec(
			service.insertIntoSchedules, // "INSERT INTO schedules VALUES($1, $2, $3, $4, $5, $6, $7, $8)"
			candidate.GUID,
			candidate.Enabled,
			candidate.Expression,
			candidate.ExpressionType,
			candidate.RefGUID,
			candidate.RefType,
			candidate.CreatedAt,
			candidate.UpdatedAt,
			nullable(candidate.PausedBy),
		)

		return aErr
	})

	if err != nil {
		return nil, err
	}

	return candidate, nil
}

func (service *ScheduleService) getCollection(query string, args ...any) []*core.Schedule {
	collection := make([]*core.Schedule, 0)

	found, _ := service.query(query, args...)

	return append(collection, found...)
}

// query runs a schedule lookup and returns the error from the query or from
// reading its rows.
func (service *ScheduleService) query(query string, args ...any) ([]*core.Schedule, error) {
	rows, err := service.db.Query(query, args...)
	if err != nil {
		return nil, err
	}

	return scanSchedules(rows)
}

// scanSchedules reads scheduleColumns rows and closes them. It returns every
// row that scans, along with the first error from a row that doesn't scan or
// from the rows themselves. Callers that must not mistake a damaged row for a
// missing one check the error, and getCollection still takes the good rows.
func scanSchedules(rows *sql.Rows) ([]*core.Schedule, error) {
	defer rows.Close()

	collection := make([]*core.Schedule, 0)

	var scanErr error

	for rows.Next() {
		var guid string
		var enabled bool
		var expression string
		var expressionType string
		var refGUID string
		var refType string
		var createdAt time.Time
		var updatedAt time.Time
		var pausedBy sql.NullString

		err := rows.Scan(&guid, &enabled, &expression, &expressionType, &refGUID, &refType, &createdAt, &updatedAt, &pausedBy)
		if err != nil {
			if scanErr == nil {
				scanErr = err
			}

			continue
		}

		candidate := &core.Schedule{
			GUID:           guid,
			Enabled:        enabled,
			Expression:     expression,
			ExpressionType: expressionType,
			RefGUID:        refGUID,
			RefType:        refType,
			CreatedAt:      createdAt,
			UpdatedAt:      updatedAt,
			PausedBy:       pausedBy.String,
		}

		collection = append(collection, candidate)
	}

	if scanErr != nil {
		return collection, scanErr
	}

	return collection, rows.Err()
}
