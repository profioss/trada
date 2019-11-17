package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/profioss/clog"
)

// App defines application.
type App struct {
	Config
	client  *http.Client
	log     clog.Logger
	logFile *os.File
}

// Close finishes App by closing open resources.
func (a *App) Close() {
	a.client.CloseIdleConnections()
	a.logFile.Close()
}

// Validate checks if App is valid.
func (a App) Validate() error {
	switch {
	case a.Config.Validate() != nil:
		return fmt.Errorf("config validation error: %s", a.Config.Validate())

	case a.client == nil:
		return fmt.Errorf("client is not initialized")

	case a.log == nil:
		return fmt.Errorf("log is not initialized")
	}

	return nil
}

// newApp creates new App.
func newApp() (App, error) {
	app := App{}

	conf, err := initConfig(initSettings())
	if err != nil {
		return app, err
	}
	app.Config = conf

	app.client = mkClient(conf)

	logf, err := clog.OpenFile(conf.Setup.LogFile)
	if err != nil {
		return app, err
	}
	logger, err := clog.New(logf, conf.Setup.LogLevel, conf.verbose)
	if err != nil {
		return app, err
	}
	app.log = logger

	return app, app.Validate()
}

func mkClient(conf Config) *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			DisableKeepAlives: false,
		},
		Timeout: time.Duration(conf.Setup.Timeout),
	}
}
