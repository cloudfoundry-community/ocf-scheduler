package postgres

import (
    "testing"
    "time"

    "github.com/cloudfoundry-community/ocf-scheduler/core"
    "github.com/ess/testscope"
)

// helper to create and insert a schedule directly (mirrors dummyCall pattern)
func dummySchedule(s *core.Schedule) *core.Schedule {
    if s == nil { s = &core.Schedule{} }

    now := time.Now().UTC()
    s.CreatedAt = now
    s.UpdatedAt = now

    if len(s.GUID) == 0 { s.GUID, _ = core.GenGUID() }
    if len(s.Expression) == 0 { s.Expression = "* * * * *" }
    if len(s.ExpressionType) == 0 { s.ExpressionType = "cron" }
    if len(s.RefGUID) == 0 { s.RefGUID, _ = core.GenGUID() }
    if len(s.RefType) == 0 { s.RefType = "call" }

    WithTransaction(testdb, func(tx Transaction) error {
        tx.Exec(
            "INSERT INTO schedules VALUES($1,$2,$3,$4,$5,$6,$7,$8)",
            s.GUID,
            s.Enabled,
            s.Expression,
            s.ExpressionType,
            s.RefGUID,
            s.RefType,
            s.CreatedAt,
            s.UpdatedAt,
        )
        return nil
    })

    return s
}

func TestScheduleService_PersistAndGet(t *testing.T) {
    testscope.SkipUnlessUnit(t)
    service := NewScheduleService(testdb)
    Cleaner.Acquire("schedules")

    candidate := &core.Schedule{Enabled: true, Expression: "*/5 * * * *", ExpressionType: "cron", RefType: "call"}
    candidate.RefGUID, _ = core.GenGUID()

    persisted, err := service.Persist(candidate)

    if err != nil { t.Fatalf("unexpected error persisting schedule: %v", err) }
    if persisted.GUID == "" { t.Errorf("expected GUID to be set") }

    fetched, gErr := service.Get(persisted.GUID)
    if gErr != nil { t.Errorf("expected Get to succeed, got %v", gErr) }
    if fetched == nil || fetched.GUID != persisted.GUID { t.Errorf("fetched schedule mismatch") }

    Cleaner.Clean("schedules")
}

func TestScheduleService_Get_NotFound(t *testing.T) {
    testscope.SkipUnlessUnit(t)
    service := NewScheduleService(testdb)
    Cleaner.Acquire("schedules")
    guid, _ := core.GenGUID()
    got, err := service.Get(guid)
    if got != nil { t.Errorf("expected nil schedule") }
    if err == nil { t.Errorf("expected error for missing schedule") }
    Cleaner.Clean("schedules")
}

func TestScheduleService_Delete(t *testing.T) {
    testscope.SkipUnlessUnit(t)
    service := NewScheduleService(testdb)
    Cleaner.Acquire("schedules")
    s := dummySchedule(&core.Schedule{Enabled: true})
    if err := service.Delete(&core.Schedule{GUID: s.GUID}); err != nil { t.Errorf("delete returned error: %v", err) }
    // ensure it is gone
    got, _ := service.Get(s.GUID)
    if got != nil { t.Errorf("expected schedule to be deleted") }
    Cleaner.Clean("schedules")
}

func TestScheduleService_ByCall_And_ByJob(t *testing.T) {
    testscope.SkipUnlessUnit(t)
    service := NewScheduleService(testdb)
    Cleaner.Acquire("schedules")
    callGUID, _ := core.GenGUID()
    jobGUID, _ := core.GenGUID()
    dummySchedule(&core.Schedule{RefType: "call", RefGUID: callGUID})
    dummySchedule(&core.Schedule{RefType: "job", RefGUID: jobGUID})

    callSchedules := service.ByCall(&core.Call{GUID: callGUID})
    if len(callSchedules) != 1 { t.Errorf("expected 1 call schedule, got %d", len(callSchedules)) }
    jobSchedules := service.ByJob(&core.Job{GUID: jobGUID})
    if len(jobSchedules) != 1 { t.Errorf("expected 1 job schedule, got %d", len(jobSchedules)) }
    Cleaner.Clean("schedules")
}

func TestScheduleService_Enabled(t *testing.T) {
    testscope.SkipUnlessUnit(t)
    service := NewScheduleService(testdb)
    Cleaner.Acquire("schedules")
    dummySchedule(&core.Schedule{Enabled: true})
    dummySchedule(&core.Schedule{Enabled: false})
    enabled := service.Enabled()
    if len(enabled) != 1 { t.Errorf("expected 1 enabled schedule, got %d", len(enabled)) }
    if !enabled[0].Enabled { t.Errorf("expected schedule to be enabled") }
    Cleaner.Clean("schedules")
}
