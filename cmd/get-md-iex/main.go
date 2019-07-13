package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/profioss/trada/pkg/osutil"
)

func main() {
	exitCode := 0
	wg := sync.WaitGroup{}

	app, err := newApp()
	if err != nil {
		log.Fatal("App init error: ", err)
	}
	defer app.Close()

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

	err = os.MkdirAll(app.Setup.OutputDir, osutil.DirPerms)
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
	return getData(ctx, app)
}

func cleanup(app App) {
	app.log.Info("Cleaning up...")
}
