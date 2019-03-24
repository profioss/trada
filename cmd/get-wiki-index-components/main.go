package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
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

func main() {
	exitCode := 0
	wg := sync.WaitGroup{}

	app, err := NewApp()
	if err != nil {
		log.Fatal("App init error: ", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)

	// common cleanup
	defer func() {
		signal.Stop(sigChan)
		cancel()
		cleanup(app)
		wg.Wait()
		os.Exit(exitCode)
	}()

	go func() {
		wg.Add(1)
		select {
		case s := <-sigChan:
			app.log.Warnf("Got %s signal - exitting", s)
			cancel()
			app.log.Info("Stopped")
		case <-ctx.Done():
			app.log.Info("DONE")
		}
		wg.Done()
	}()

	err = os.MkdirAll(app.Setup.OutputDir, os.FileMode(0755))
	if err != nil {
		exitCode = 1
		app.log.Errorf("Prepare OutputDir error: %s", err)
		return
	}

	app.log.Info("Starting")

	err = do(ctx, app)
	if err != nil {
		exitCode = 1
		app.log.Errorf("Fetch data error: %s", err)
		return
	}
}

func do(ctx context.Context, app App) error {
	for _, r := range app.Resources {
		app.log.Info("Fetching ", r.Name)
		getNparse(ctx, app, r)
	}

	return nil
}

func getNparse(ctx context.Context, app App, ds DataSrc) error {

	return nil
}

func mkClient(conf Config) *http.Client {
	tr := &http.Transport{DisableKeepAlives: false}
	timeout := time.Duration(conf.Setup.Timeout)
	return &http.Client{Transport: tr, Timeout: timeout}
}

func mkURL(conf Config, ds DataSrc) (url.URL, error) {
	u, err := url.Parse(conf.Setup.WikiAPI)
	if err != nil {
		return url.URL{}, err
	}

	q := u.Query()
	q.Set("action", "parse")
	q.Set("format", "json")
	q.Set("prop", "text")
	q.Set("page", ds.PageName)
	q.Set("section", strconv.Itoa(ds.Section))
	u.RawQuery = q.Encode()

	return *u, nil
}

func cleanup(app App) {
	app.log.Info("Cleaning up...")
}
