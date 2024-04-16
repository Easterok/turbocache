package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/easterok/turbocache/pkg/storage"
	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
)

type (
	Meta struct {
		Hash          string `json:"hash"`
		ContentLength string `json:"content-length"`
		Duration      string `json:"duration"`
	}

	Event struct {
		SessionId string `json:"sessionId"`
		Source    string `json:"source"`
		Event     string `json:"event"`
		Hash      string `json:"hash"`
		Duration  int    `json:"duration"`
	}
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

	artifacts.GET("/info", func(c echo.Context) error {
		team := c.QueryParam("teamid")

		if team == "" {
			team = c.QueryParam("slug")
		}

		if team == "" {
			return c.String(http.StatusBadRequest, "querystring should have required property teamid or slug")
		}

		meta, err := storage.GetMeta(team)

		if err != nil {
			return c.JSON(http.StatusOK, Info{
				CorruptedData: true,
			})
		}

		events, err := storage.GetEvents(team)

		if err != nil {
			return c.JSON(http.StatusOK, Info{
				CorruptedData: true,
			})
		}

		status := prepareInfo(string(meta), string(events))

		return c.JSON(http.StatusOK, status)
	})

	artifacts.GET("/status", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]string{"status": "enabled"})
	})

	artifacts.POST("/events", func(c echo.Context) error {
		team := c.QueryParam("teamid")

		if team == "" {
			team = c.QueryParam("slug")
		}

		if team == "" {
			return c.String(http.StatusBadRequest, "querystring should have required property teamid or slug")
		}

		value := []Event{}

		err := json.NewDecoder(c.Request().Body).Decode(&value)

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		}

		m, err := json.Marshal(value)

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		}

		err = storage.SaveEvent(team, m)

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		}

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

		duration := c.Request().Header.Get("x-artifact-duration")
		contentLength := c.Request().Header.Get("Content-Length")

		err := storage.Put(hash, team, c.Request().Body)

		if err != nil {
			return c.String(http.StatusBadRequest, fmt.Sprintf("%v", err))
		}

		meta, _ := json.Marshal(Meta{
			Duration:      duration,
			ContentLength: contentLength,
			Hash:          hash,
		})

		err = storage.SaveMeta(hash, team, meta)

		if err != nil {
			fmt.Printf("error saving to meta file %v", err)
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

type Info struct {
	CorruptedData    bool   `json:"corrupted_data"`
	SavedTime        string `json:"saved_time"`
	TotalComputeTime string `json:"total_compute_time"`
	ContentLength    string `json:"content_length"`
}

func prepareInfo(meta, events string) Info {
	_meta := []Meta{}

	err := json.Unmarshal([]byte("["+strings.Join(strings.Split(strings.TrimSpace(meta), "\n"), ",")+"]"), &_meta)

	if err != nil {
		return Info{
			CorruptedData: true,
		}
	}

	_events := [][]Event{}

	err = json.Unmarshal([]byte("["+strings.Join(strings.Split(strings.TrimSpace(events), "\n"), ",")+"]"), &_events)

	if err != nil {
		return Info{
			CorruptedData: true,
		}
	}

	duration := 0
	saved := 0
	contentLength := 0

	for _, item := range _meta {
		a, _ := strconv.Atoi(item.ContentLength)

		contentLength += a
	}

	corrupted_data := false

	remote := []Event{}

	for _, i := range _events {
		for _, event := range i {
			if event.Source == "REMOTE" {
				remote = append(remote, event)
			}
		}
	}

	for _, event := range remote {
		founded := false

		for _, item := range _meta {
			if !founded && item.Hash == event.Hash {
				_duration, _ := strconv.Atoi(item.Duration)

				if event.Event == "HIT" {
					saved += _duration
				} else {
					duration += _duration
				}

				founded = true
			}

			duration += event.Duration
		}

		corrupted_data = !founded || corrupted_data
	}

	return Info{
		CorruptedData:    corrupted_data,
		SavedTime:        fmt.Sprintf("%d (ms)", saved),
		TotalComputeTime: fmt.Sprintf("%d (ms)", duration),
		ContentLength:    fmt.Sprintf("%d (bytes)", contentLength),
	}
}
