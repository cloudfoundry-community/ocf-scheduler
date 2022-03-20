package auth

import (
	"fmt"

	"github.com/labstack/echo"
)

func Verify(c echo.Context) error {
	auth := c.Request().Header.Get(echo.HeaderAuthorization)

	return fmt.Errorf("unimplemented")
}
