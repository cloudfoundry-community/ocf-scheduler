package core

import "time"

// Finding is one error or warning about a cron expression. Code is stable
// for scripts; Message is for people.
type Finding struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Field      string `json:"field,omitempty"`
	Value      string `json:"value,omitempty"`
	ValidRange string `json:"valid_range,omitempty"`
	Offset     *int   `json:"offset,omitempty"` // byte offset of Value in the expression
}

// HashedField is the value an H in one field resolved to.
type HashedField struct {
	Field    string `json:"field"`
	Value    string `json:"value"`
	Resolved string `json:"resolved"`
}

// Ref names the job or call an expression was validated for.
type Ref struct {
	Type string `json:"type"`
	GUID string `json:"guid"`
	Name string `json:"name"`
}

// ScheduleAnalysis is what the scheduler would do with one cron expression.
type ScheduleAnalysis struct {
	Expression   string        `json:"expression"`
	Ref          *Ref          `json:"ref,omitempty"`
	ScheduleGUID string        `json:"schedule_guid,omitempty"`
	Enabled      *bool         `json:"enabled,omitempty"`
	Valid        bool          `json:"valid"`
	Description  string        `json:"description"`
	Location     string        `json:"location,omitempty"`
	Illustrative bool          `json:"illustrative"`
	HashedFields []HashedField `json:"hashed_fields"`
	NextRuns     []time.Time   `json:"next_runs"`
	PrevRuns     []time.Time   `json:"prev_runs"`
	Errors       []Finding     `json:"errors"`
	Warnings     []Finding     `json:"warnings"`
}

// ValidateRequest is the body of POST /schedules/validate.
type ValidateRequest struct {
	Expression string `json:"expression"`
	RefType    string `json:"ref_type"`
	RefGUID    string `json:"ref_guid"`
	Next       *int   `json:"next"`
	Prev       int    `json:"prev"`
}

// Findings is the body of a 4xx response about a cron expression or request.
type Findings struct {
	Errors   []Finding `json:"errors"`
	Warnings []Finding `json:"warnings"`
}
