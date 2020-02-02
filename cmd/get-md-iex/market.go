package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"

	"github.com/profioss/trada/model/instrument"
	"github.com/profioss/trada/model/ohlc"
	"github.com/profioss/trada/model/ohlc/ohlcio"
	"github.com/profioss/trada/pkg/osutil"
)

func work(ctx context.Context, app App) error {
	var err error
	instruments := app.Config.instrSpecs // specified by command line param
	if len(instruments) == 0 {           // no symbol specified by command line param
		instruments, err = loadInstruments(app) // load from watchlist(s)
		if err != nil {
			return fmt.Errorf("loadTickers failed %s", err)
		}
	}

	switch {
	case len(instruments) == 0:
		return fmt.Errorf("empty instrument list")

	case len(instruments) < app.Config.Setup.MaxProcs && len(instruments) > 0:
		app.Config.Setup.MaxProcs = len(instruments)
	}

	for _, spec := range instruments {
		select {
		case <-ctx.Done():
			return fmt.Errorf("operation cancelled")
		default:
		}
		fname, err := getNstore(ctx, app, spec)
		if err != nil {
			app.log.Errorf("%s: %v", spec.Symbol, err)
			continue
		}
		app.log.Infof("%s: saved to %s", spec.Symbol, fname)
	}

	return nil
}

func getNstore(ctx context.Context, app App, spec instrument.Spec) (string, error) {
	select {
	case <-ctx.Done():
		return "", fmt.Errorf("operation cancelled")
	default:
	}

	data, err := fetch(ctx, spec, app)
	if err != nil {
		app.log.Errorf("%s: fetch error: %s", spec.Symbol, err)
		return "", fmt.Errorf("%s: fetch error: %s", spec.Symbol, err)
	}
	app.log.Debugf("%s: fetch - OK", spec.Symbol)

	fname := filepath.Join(app.Config.Setup.OutputDir, spec.Symbol)

	dataOHLC := []ohlc.OHLC{}
	if app.Config.Setup.Range == "1d" {
		ohlc := ohlc.OHLC{}
		err = json.Unmarshal(data, &ohlc)
		dataOHLC = append(dataOHLC, ohlc)
	} else {
		err = json.Unmarshal(data, &dataOHLC)
	}
	// store problematic data to .../dir/fname.json.swp for analysis
	fnameFetch := fname + ".json.swp"
	if err != nil {
		osutil.WriteFile(fnameFetch, data)
		return "", fmt.Errorf("%s: unmarshal error: %s; check %s",
			spec.Symbol, err, fnameFetch)
	}
	// clean up after possible previous errors
	os.Remove(fnameFetch)

	fname += ".csv"
	dataCSV := ohlcio.ToCSV(dataOHLC, spec.SecurityType)
	err = saveData(ctx, fname, dataCSV)
	if err != nil {
		return fname, fmt.Errorf("%s: saveData error: %s", spec.Symbol, err)
	}

	return fname, nil
}

func fetch(ctx context.Context, spec instrument.Spec, app App) ([]byte, error) {
	output := []byte{}
	select {
	case <-ctx.Done():
		return output, fmt.Errorf("operation cancelled")
	default:
	}

	url, err := mkURL(app.Config, spec.Symbol)
	if err != nil {
		return output, fmt.Errorf("mkUrl failed: %v", err)
	}

	resp, err := app.client.Get(url.String())
	if err != nil {
		return output, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return output, fmt.Errorf("fetch: HTTP Status: %s. URL: %s", resp.Status, url.String())
	}

	output, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return output, fmt.Errorf("fetch: Read Error: %s", err)
	}

	return output, nil
}

func mkURL(conf Config, ticker string) (url.URL, error) {
	str := fmt.Sprintf("%s/stock/%s/chart/%s",
		conf.Setup.BaseURL, ticker, conf.Setup.Range)
	if conf.Setup.Range == "1d" {
		str = fmt.Sprintf("%s/stock/%s/previous", conf.Setup.BaseURL, ticker)
	}

	u, err := url.Parse(str)
	if err != nil {
		return url.URL{}, err
	}

	q := u.Query()
	q.Set("token", conf.Setup.Token)
	q.Set("format", "json")
	u.RawQuery = q.Encode()

	return *u, nil
}

func saveData(ctx context.Context, fpath string, data [][]string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled")
	default:
	}

	output := data
	err := osutil.FileExists(fpath)
	if err == nil {
		dataMerged, err := mergeData(fpath, data)
		if err != nil {
			return fmt.Errorf("mergeData %s failed: %v", fpath, err)
		}
		output = dataMerged
	}

	err = writeData(ctx, fpath, output)
	if err != nil {
		return fmt.Errorf("writeData %s failed: %v", fpath, err)
	}

	return nil
}

func mergeData(fpath string, dataNew [][]string) ([][]string, error) {
	header := dataNew[0]
	output := [][]string{header}
	data := map[string][]string{}

	file, err := os.Open(fpath)
	if err != nil {
		return output, fmt.Errorf("open data file error: %v", err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Comma = ';'
	_, _ = reader.Read() // read CSV header
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			return output, fmt.Errorf("read data file error: %v", err)
		}
		data[record[0]] = record
	}

	for _, record := range dataNew[1:] { // skip CSV header
		data[record[0]] = record
	}

	sorted := make([][]string, 0, len(data))
	for _, record := range data {
		sorted = append(sorted, record)
	}
	// sort by date string
	sort.Slice(sorted,
		func(i, j int) bool { return sorted[i][0] < sorted[j][0] })

	output = append(output, sorted...)
	return output, nil
}

func writeData(ctx context.Context, fpath string, data [][]string) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("operation cancelled")
	default:
	}

	dirname := filepath.Dir(fpath)
	err := os.MkdirAll(dirname, osutil.DirPerms)
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
		return fmt.Errorf("CSV temp file error: %s", w.Error())
	}

	err = os.Chmod(fdTmp.Name(), osutil.FilePerms)
	if err != nil {
		return fmt.Errorf("chmod %s %s: %s", osutil.FilePerms.String(), fdTmp.Name(), err)
	}
	err = os.Rename(fdTmp.Name(), fpath)
	if err != nil {
		return fmt.Errorf("rename %s -> %s: %s", fdTmp.Name(), fpath, err)
	}

	return nil
}

func loadInstruments(app App) ([]instrument.Spec, error) {
	funcName := "loadInstruments"
	output := []instrument.Spec{}
	// specMap is unique (key: symbol + security type) instrument collection
	specMap := make(map[string]instrument.Spec)

	for _, path := range app.Config.Setup.Watchlists {
		fd, err := os.Open(path)
		if err != nil {
			return output, fmt.Errorf("open %s error: %s", path, err)
		}
		defer fd.Close()

		specLst, err := instrument.SpecLstFromCSV(fd)
		if err != nil {
			return output, fmt.Errorf("load from %s error: %s", path, err)
		}
		if len(specLst) == 0 {
			app.log.Warnf("%s: %s is empty", funcName, path)
		}

		for _, s := range specLst {
			specMap[s.Symbol+s.SecurityType.String()] = s
		}
		app.log.Infof("%s: %s - OK", funcName, path)
	}

	for _, spec := range specMap {
		output = append(output, spec)
	}
	app.log.Infof("%s: loaded %d instruments", funcName, len(output))

	return output, nil
}
