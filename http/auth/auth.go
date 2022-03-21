package auth

import (
	"fmt"

	"github.com/labstack/echo/v4"
)

func Verify(c echo.Context) error {
	auth := c.Request().Header.Get(echo.HeaderAuthorization)
	if len(auth) == 0 {
		return fmt.Errorf("no auth provided")
	}

	if auth != "jeremy" {
		return fmt.Errorf("you're not jeremy!")
	}

	return nil
}
