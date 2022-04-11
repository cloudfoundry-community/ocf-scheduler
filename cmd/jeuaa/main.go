package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	uaa "github.com/cloudfoundry-community/go-uaa"
	"github.com/labstack/echo/v4"
)

func Server(bind string) *http.Server {
	e := echo.New()

	// not technically necessary, just part of my default API skeleton
	e.GET("/userinfo", func(c echo.Context) error {
		auth := c.Request().Header.Get(echo.HeaderAuthorization)

		parts := strings.Split(auth, " ")
		if parts[0] != "Bearer" {
			return c.JSON(
				http.StatusUnauthorized,
				"",
			)
		}

		if parts[1] != "jeremy" {
			return c.JSON(
				http.StatusUnauthorized,
				"",
			)
		}

		userinfo := &uaa.UserInfo{
			//UserID:            "",
			Sub:         "12345",
			Username:    "dummy",
			GivenName:   "Dummy",
			FamilyName:  "User",
			Email:       "dummy@example.com",
			PhoneNumber: "",
			//PreviousLoginTime: time.Now().UTC().Unix(),
			Name: "Dummy User",
		}

		//fmt.Println("payload:", string(payload))

		return c.JSON(
			http.StatusOK,
			userinfo,
		)
	})

	e.POST("/oauth/token", func(c echo.Context) error {
		return c.JSON(
			http.StatusOK,
			map[string]interface{}{
				"token_type":    "bearer",
				"access_token":  "jeremy",
				"refresh_token": "bearamy",
			},
		)
	})

	e.GET("*", func(c echo.Context) error {
		fmt.Println("Got a GET request I didn't recognize:", c.Request().URL)

		return c.JSON(http.StatusInternalServerError, "")
	})

	e.POST("*", func(c echo.Context) error {
		fmt.Println("Got a POST request I didn't recognize:", c.Request().URL)

		return c.JSON(http.StatusInternalServerError, "")
	})

	server := e.Server
	server.Addr = bind

	return server
}

func main() {
	server := Server("0.0.0.0:8001")

	go func() {
		if err := server.ListenAndServe(); err != nil {
			fmt.Println("stopping the server")
		}
	}()

	fmt.Printf("listening for connections on %s\n", server.Addr)

	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt)

	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		server.Close()
		fmt.Println(err.Error())
		os.Exit(2)
	}
}
