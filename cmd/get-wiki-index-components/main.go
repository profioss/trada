package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

const dirPerms os.FileMode = 0755
const filePerms os.FileMode = 0644

type parser = func(r io.Reader) ([][]string, error)

// Mapping of DataSrc.Name in Config with content parser.
// NOTE: this is validated - using proper names is required.
var parsers = map[string]parser{
	"DJIA": parseDJIA,
	"OEX":  parseOEX,
	"SPX":  parseSPX,
	"NDX":  parseNDX,
}

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
		getNparse(ctx, app, r)
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

	wd, err := parseWikiData(fnameSrc)
	if err != nil {
		return err
	}

	components, err := parseData(app, wd, ds)
	if err != nil {
		app.log.Errorf("%s: parseData failed: %s", ds.Name, err)
		return err
	}
	app.log.Debugf("%s: parseData - OK", ds.Name)

	data := [][]string{
		[]string{"sym", "name"}, // CSV output header
	}
	data = append(data, components...)

	fnameDst := filepath.Join(app.Setup.OutputDir, ds.OutputFile)
	err = saveData(fnameDst, data)
	if err != nil {
		app.log.Errorf("%s: saveData to %s failed: %s", ds.Name, fnameDst, err)
		return err
	}
	app.log.Debugf("%s: saveData to %s - OK", ds.Name, fnameDst)

	app.log.Infof("%s: %s - OK", ds.Name, fnameDst)
	return nil
}

func parseData(app App, wd wikiData, ds DataSrc) ([][]string, error) {
	p, ok := parsers[ds.Name]
	if !ok {
		msg := fmt.Sprintf("No parser for %s", ds.Name)
		app.log.Error(msg)
		return [][]string{}, fmt.Errorf(msg)
	}

	wdr := strings.NewReader(wd.Parsed.Content.Text)
	components, err := p(wdr)
	if err != nil {
		app.log.Errorf("%s: parsing table failed: %s", ds.Name, err)
		return [][]string{}, err
	}

	return components, nil
}

func saveData(fpath string, data [][]string) error {
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

	w := csv.NewWriter(fdTmp)
	w.Comma = ';'
	w.WriteAll(data)
	if w.Error() != nil {
		return fmt.Errorf("temp file error: %s", w.Error())
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
