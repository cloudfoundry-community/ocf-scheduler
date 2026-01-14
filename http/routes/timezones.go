package routes

import (
	"fmt"
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/cloudfoundry-community/ocf-scheduler/core"
)

type timezoneCollection struct {
	Pagination *pagination         `json:"pagination"`
	Resources  *core.TimezoneSlice `json:"resources"`
}

func Timezones(e *echo.Echo, services *core.Services) {
	// Get server timezones
	// GET /timezones
	e.GET("/scheduler-time-zones", func(c echo.Context) error {
		tag := "timezones"
		services.Logger.Info(tag, "trying to get timezones")

		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		if services.Auth.Verify(auth) != nil {
			services.Logger.Error(tag, "authentication to this endpoint failed")
			return c.JSON(http.StatusUnauthorized, "")
		}

		timezones, err := services.Cron.GetTimezones()
		if err != nil {
			services.Logger.Error(tag, fmt.Sprintf("error retrieving timezone: %v", err))
			return c.JSON(http.StatusInternalServerError, "error retrieving timezones")
		}

		output := &timezoneCollection{
			Resources: timezones,
			Pagination: &pagination{
				TotalPages:   1,
				TotalResults: len(*timezones),
				First:        &pageref{Href: "first"},
				Last:         &pageref{Href: "last"},
				Next:         &pageref{Href: "next"},
				Previous:     &pageref{Href: "previous"},
			},
		}

		return c.JSON(
			http.StatusOK,
			output,
		)
	})
}
