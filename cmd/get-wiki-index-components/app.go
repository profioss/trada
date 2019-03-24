package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/profioss/clog"
)

// App defines application.
type App struct {
	Config
	client *http.Client
	log    clog.Logger
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

// NewApp creates new App.
func NewApp() (App, error) {
	app := App{}

	conf, err := initConfig()
	if err != nil {
		return app, err
	}
	app.Config = conf

	app.client = mkClient(conf)

	logf, err := clog.OpenFile(conf.Setup.LogFile)
	if err != nil {
		return app, err
	}
	defer logf.Close()
	logger, err := clog.New(logf, conf.Setup.LogLevel, conf.verbose)
	if err != nil {
		return app, err
	}
	app.log = logger

	return app, app.Validate()
}

func mkClient(conf Config) *http.Client {
	tr := &http.Transport{DisableKeepAlives: false}
	timeout := time.Duration(conf.Setup.Timeout)
	return &http.Client{Transport: tr, Timeout: timeout}
}
