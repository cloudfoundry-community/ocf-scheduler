package routes

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
	"github.com/cloudfoundry-community/ocf-scheduler/cron"
	"github.com/cloudfoundry-community/ocf-scheduler/logger"
	"github.com/cloudfoundry-community/ocf-scheduler/mock"
)

type fixture struct {
	e        *echo.Echo
	services *core.Services
	job      *core.Job
	call     *core.Call
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	services := &core.Services{
		Jobs:      mock.NewJobService(),
		Calls:     mock.NewCallService(),
		Schedules: mock.NewScheduleService(),
		Cron:      cron.NewCronService(logger.New(), cron.Rules{}),
		Logger:    logger.New(),
		Auth:      mock.NewAuthService(),
	}
	job, err := services.Jobs.Persist(&core.Job{Name: "backup", AppGUID: "app-1", SpaceGUID: "space-1"})
	if err != nil {
		t.Fatal(err)
	}
	call, err := services.Calls.Persist(&core.Call{Name: "ping", AppGUID: "app-1", SpaceGUID: "space-1"})
	if err != nil {
		t.Fatal(err)
	}
	e := echo.New()
	Apply(e, services)
	return &fixture{e: e, services: services, job: job, call: call}
}

func (f *fixture) post(path, body, token string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	if token != "" {
		req.Header.Set(echo.HeaderAuthorization, token)
	}
	rec := httptest.NewRecorder()
	f.e.ServeHTTP(rec, req)
	return rec
}
