package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"

	_ "github.com/profioss/trada/cmd/get-wiki-index-components/djia"
	_ "github.com/profioss/trada/cmd/get-wiki-index-components/ndx"
	_ "github.com/profioss/trada/cmd/get-wiki-index-components/oex"
	"github.com/profioss/trada/cmd/get-wiki-index-components/parser"
	_ "github.com/profioss/trada/cmd/get-wiki-index-components/spx"
	"github.com/profioss/trada/model/instrument"
	"github.com/profioss/trada/pkg/wiki"
)

const dirPerms os.FileMode = 0755
const filePerms os.FileMode = 0644

/*
type parser = func(r io.Reader) ([]instrument.Spec, error)

// Mapping of DataSrc.Name in Config with content parser.
// NOTE: this is validated - using proper names is required.
var parsers = map[string]parser{
	// "DJIA": parseDJIA,
	"OEX": parseOEX,
	"SPX": parseSPX,
	"NDX": parseNDX,
}
*/

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

	err = os.MkdirAll(app.Setup.OutputDir, dirPerms)
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
		err := getNparse(ctx, app, r)
		if err != nil {
			app.log.Error(err)
			// don't stop - get next resource
		}
	}

	return nil
}

func getNparse(ctx context.Context, app App, ds DataSrc) error {
	fnameSrc, err := getData(ctx, app, ds)
	if err != nil {
		app.log.Errorf("%s: getData failed: %s. Leaving file %s", ds.Name, err, fnameSrc)
		return err
	}
	app.log.Debugf("%s: getData to %s - OK", ds.Name, fnameSrc)

	if !app.updateTstData { // update testing data mode
		defer os.Remove(fnameSrc)
	}

	fd, err := os.Open(fnameSrc)
	if err != nil {
		return fmt.Errorf("open %s error: %v", fnameSrc, err)
	}
	defer fd.Close()
	wd, err := wiki.Parse(fd)
	if err != nil {
		return err
	}

	components, err := parseData(app, wd, ds)
	if err != nil {
		app.log.Errorf("%s: parseData failed: %s", ds.Name, err)
		return err
	}
	app.log.Debugf("%s: parseData - OK", ds.Name)
	sort.Slice(components, func(i, j int) bool { return components[i].Symbol < components[j].Symbol })

	// don't overwrite with insufficient data
	if len(components) < ds.MinCnt {
		return fmt.Errorf("%s: expected at least %d components, got %d", ds.Name, ds.MinCnt, len(components))
	}

	fnameDst := filepath.Join(app.Setup.OutputDir, ds.OutputFile)
	err = saveData(fnameDst, components)
	if err != nil {
		app.log.Errorf("%s: saveData to %s failed: %s", ds.Name, fnameDst, err)
		return err
	}
	app.log.Debugf("%s: saveData to %s - OK", ds.Name, fnameDst)

	app.log.Infof("%s: %s - OK", ds.Name, fnameDst)
	return nil
}

func parseData(app App, wd wiki.Data, ds DataSrc) ([]instrument.Spec, error) {
	p, err := parser.Get(ds.Name)
	if err != nil {
		app.log.Error(err)
		return []instrument.Spec{}, err
	}

	wdr := strings.NewReader(wd.Parsed.Content.Text)
	components, err := p.Parse(wdr)
	if err != nil {
		app.log.Errorf("%s: parsing table failed: %s", ds.Name, err)
		return []instrument.Spec{}, err
	}

	return components, nil
}

func saveData(fpath string, components []instrument.Spec) error {
	dirname := filepath.Dir(fpath)
	err := os.MkdirAll(dirname, dirPerms)
	if err != nil {
		return err
	}

	fdTmp, err := os.Create(fpath + ".swp")
	if err != nil {
		return fmt.Errorf("creating temp output file failed: %s", err)
	}
	defer os.Remove(fdTmp.Name())

	err = instrument.SpecLstToCSV(fdTmp, components)
	if err != nil {
		return fmt.Errorf("instrument.SpecLstToCSV: %v", err)
	}

	err = os.Chmod(fdTmp.Name(), filePerms)
	if err != nil {
		return fmt.Errorf("chmod %s %s: %s", filePerms.String(), fdTmp.Name(), err)
	}
	err = os.Rename(fdTmp.Name(), fpath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %s", fdTmp.Name(), fpath, err)
	}

	return nil
}

func getData(ctx context.Context, app App, ds DataSrc) (string, error) {
	u, err := mkURL(app.Config, ds)
	if err != nil {
		return "", err
	}

	resp, err := app.client.Get(u.String())
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// TODO - check what codes the API returns - if it behaves correctly
		return "", fmt.Errorf("HTTP error: %d for %s", resp.StatusCode, u.String())
	}

	fname := filepath.Join(app.Setup.OutputDir, ds.OutputFile+".json")
	fd, err := os.Create(fname)
	if err != nil {
		return "", err
	}
	defer fd.Close()

	_, err = io.Copy(fd, resp.Body)
	if err != nil {
		return "", err
	}

	return fd.Name(), nil
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
