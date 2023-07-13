package helpers

import (
	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

func Jobify(c echo.Context) *core.Job {
	job := &core.Job{}
	c.Bind(&job)

	return job
}

func Callify(c echo.Context) *core.Call {
	call := &core.Call{}
	c.Bind(&call)

	return call
}

func Schedulify(c echo.Context) *core.Schedule {
	schedule := &core.Schedule{}
	c.Bind(&schedule)

	return schedule
}

func Executionify(c echo.Context) *core.Execution {
	exe := &core.Execution{}
	c.Bind(&exe)

	return exe
}
