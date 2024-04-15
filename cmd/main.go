package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/easterok/turbocache/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

func main() {
	e := echo.New()

	err := godotenv.Load()

	if err != nil {
		log.Fatal("Error loading .env file")
	}

	availableTokens := strings.Split(os.Getenv("TURBO_TOKENS"), ",")

	if len(availableTokens) == 0 {
		log.Fatal("missing TURBO_TOKENS in .env")
	}

	e.Use(middleware.Logger())

	tmpdir := os.Getenv("TURBO_TEMP_DIR")

	if tmpdir == "" {
		tmpdir = "turbo.go.remote-cache"
	}

	version := os.Getenv("TURBO_API_VERSION")

	if version == "" {
		version = "v8"
	}

	storage, err := storage.MakeDisk(filepath.Join(os.TempDir(), tmpdir))

	if err != nil {
		log.Fatal(err)
	}

	artifacts := e.Group(fmt.Sprintf("/%s/artifacts", version))

	artifacts.Use(middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
		KeyLookup:  "header:Authorization",
		AuthScheme: "Bearer",
		Validator: func(auth string, c echo.Context) (bool, error) {
			available := false

			for _, i := range availableTokens {
				available = available || auth == i
			}

			return available, nil
		},
	}))

	artifacts.GET("/status", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "enabled"})
	})

	artifacts.POST("/events", func(c echo.Context) error {
		return c.String(http.StatusOK, "Success. Event recorded.")
	})

	artifacts.HEAD("/:hash", func(c echo.Context) error {
		hash := c.Param("hash")
		team := c.QueryParam("teamid")

		if team == "" {
			team = c.QueryParam("slug")
		}

		if team == "" {
			return c.String(http.StatusBadRequest, "querystring should have required property teamid or slug")
		}

		_, err := storage.Get(hash, team)

		if err != nil {
			return c.String(http.StatusNotFound, fmt.Sprintf("The artifact was not found %v", err))
		}

		return c.String(http.StatusOK, "The artifact was found and headers are returned")
	})

	artifacts.PUT("/:hash", func(c echo.Context) error {
		hash := c.Param("hash")
		team := c.QueryParam("teamId")

		if team == "" {
			team = c.QueryParam("slug")
		}

		if team == "" {
			return c.String(http.StatusBadRequest, "querystring should have required property teamid or slug")
		}

		err := storage.Put(hash, team, c.Request().Body)

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		}

		return c.String(http.StatusAccepted, "ok")
	})

	artifacts.GET("/:hash", func(c echo.Context) error {
		hash := c.Param("hash")
		team := c.QueryParam("teamId")

		if team == "" {
			team = c.QueryParam("slug")
		}

		if team == "" {
			return c.String(http.StatusBadRequest, "querystring should have required property teamid or slug")
		}

		file, err := storage.Get(hash, team)

		if err != nil {
			return c.String(http.StatusNotFound, fmt.Sprintf("The artifact was not found %v", err))
		}

		return c.Blob(http.StatusOK, "application/octet-stream", file)
	})

	log.Fatal(e.Start(fmt.Sprintf(":%s", os.Getenv("API_PORT"))))
}
