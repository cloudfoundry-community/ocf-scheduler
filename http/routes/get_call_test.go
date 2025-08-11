package routes

import (
    "encoding/json"
    "errors"
    "net/http"
    "net/http/httptest"
    "testing"
    "time"

    "github.com/labstack/echo/v4"

    "github.com/cloudfoundry-community/ocf-scheduler/core"
    "github.com/cloudfoundry-community/ocf-scheduler/mock"
)

// stubCallService implements core.CallService for handler tests
type stubCallService struct {
    call *core.Call
    err  error
}

func (s *stubCallService) Get(guid string) (*core.Call, error) { return s.call, s.err }
func (s *stubCallService) Delete(*core.Call) error             { return nil }
func (s *stubCallService) Named(string) (*core.Call, error)    { return nil, errors.New("not impl") }
func (s *stubCallService) Persist(*core.Call) (*core.Call, error) { return nil, errors.New("not impl") }
func (s *stubCallService) InSpace(string) []*core.Call         { return []*core.Call{} }

// minimal log + other services placeholders
type noopLog struct{}
func (n *noopLog) Info(tag, msg string)  {}
func (n *noopLog) Error(tag, msg string) {}
func (n *noopLog) Debug(tag, msg string) {}

// compile-time assertion that noopLog satisfies core.LogService would go here if interface present in core/logging

func newServicesForCallTest(call *core.Call, err error) *core.Services {
    return &core.Services{
        Calls:  &stubCallService{call: call, err: err},
        Auth:   mock.NewAuthService(),
        Logger: &noopLog{},
    }
}

func TestGetCall_Unauthorized_NoHeader(t *testing.T) {
    e := echo.New()
    services := newServicesForCallTest(nil, errors.New("not found"))
    GetCall(e, services)

    req := httptest.NewRequest(http.MethodGet, "/calls/abc", nil)
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rec.Code)
    }
}

func TestGetCall_Unauthorized_WrongHeader(t *testing.T) {
    e := echo.New()
    services := newServicesForCallTest(nil, errors.New("not found"))
    GetCall(e, services)

    req := httptest.NewRequest(http.MethodGet, "/calls/abc", nil)
    req.Header.Set(echo.HeaderAuthorization, "not-jeremy")
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", rec.Code)
    }
}

func TestGetCall_NotFound(t *testing.T) {
    e := echo.New()
    services := newServicesForCallTest(nil, errors.New("no records returned"))
    GetCall(e, services)

    req := httptest.NewRequest(http.MethodGet, "/calls/abc", nil)
    req.Header.Set(echo.HeaderAuthorization, "jeremy")
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusNotFound {
        t.Fatalf("expected 404, got %d", rec.Code)
    }
}

func TestGetCall_Success(t *testing.T) {
    e := echo.New()
    call := &core.Call{GUID: "abc", Name: "test", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
    services := newServicesForCallTest(call, nil)
    GetCall(e, services)

    req := httptest.NewRequest(http.MethodGet, "/calls/abc", nil)
    req.Header.Set(echo.HeaderAuthorization, "jeremy")
    rec := httptest.NewRecorder()
    e.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d", rec.Code)
    }

    var got core.Call
    if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
        t.Fatalf("invalid json: %v", err)
    }
    if got.GUID != call.GUID || got.Name != call.Name {
        t.Fatalf("unexpected call in body: %+v", got)
    }
}
