package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/profioss/clog"
)

func main() {
	exitCode := 0
	var wg sync.WaitGroup

	conf, err := initConfig()
	if err != nil {
		log.Fatal("initConfig error: ", err)
	}

	logf, err := clog.OpenFile(conf.Setup.LogFile)
	if err != nil {
		log.Fatal("Log file error: ", err)
	}
	defer logf.Close()
	logger, err := clog.New(logf, conf.Setup.LogLevel, conf.verbose)
	if err != nil {
		log.Fatal("Logger error: ", err)
	}
	conf.log = logger

	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, os.Interrupt)

	// common cleanup
	defer func() {
		signal.Stop(sigChan)
		cancel()
		cleanup(conf)
		wg.Wait()
		os.Exit(exitCode)
	}()

	go func() {
		wg.Add(1)
		select {
		case s := <-sigChan:
			conf.log.Warnf("Got %s signal - exitting", s)
			cancel()
			conf.log.Info("Stopped")
		case <-ctx.Done():
			conf.log.Info("DONE")
		}
		wg.Done()
	}()

	err = os.MkdirAll(conf.Setup.OutputDir, os.FileMode(0755))
	if err != nil {
		exitCode = 1
		conf.log.Errorf("Prepare OutputDir error: %s", err)
		return
	}

	conf.log.Info("Starting")

	err = do(ctx, conf)
	if err != nil {
		exitCode = 1
		conf.log.Errorf("Fetch data error: %s", err)
		return
	}
}

func do(ctx context.Context, conf Config) error {
	for _, r := range conf.Resources {
		conf.log.Info("Fetching ", r.Name)
	}

	return nil
}

func mkClient(conf Config) *http.Client {
	tr := &http.Transport{DisableKeepAlives: false}
	timeout := time.Duration(conf.Setup.Timeout)
	return &http.Client{Transport: tr, Timeout: timeout}
}

func cleanup(conf Config) {
	conf.log.Info("Cleaning up...")
}
