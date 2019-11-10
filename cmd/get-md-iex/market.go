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

func getData(ctx context.Context, app App) error {
	var err error
	instruments := app.Config.instrSpecs // specified by command line param
	if len(instruments) == 0 {           // no symbol specified by command line param
		instruments, err = loadInstruments(app) // load from watchlist(s)
		if err != nil {
			return fmt.Errorf("loadTickers failed %s", err)
		}
	}
	if len(instruments) < app.Config.Setup.MaxProcs && len(instruments) > 0 {
		app.Config.Setup.MaxProcs = len(instruments)
	}

	for _, spec := range instruments {
		data, err := fetch(spec, app)
		if err != nil {
			app.log.Errorf("%s: fetch error: %s", spec.Symbol, err)
			continue
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
			app.log.Errorf("%s: unmarshal error: %s; check %s",
				spec.Symbol, err, fnameFetch)
			osutil.WriteFile(fnameFetch, data)
			continue
		}
		// clean up after possible previous errors
		os.Remove(fnameFetch)

		fname += ".csv"
		// TODO - use instrument
		dataCSV := ohlcio.ToCSV(dataOHLC, instrument.Equity)
		err = saveData(fname, dataCSV)
		if err != nil {
			app.log.Errorf("%s: saveData error: %s", spec.Symbol, err)
		}
		app.log.Infof("%s: saved to %s", spec.Symbol, fname)
	}

	return nil
}

func fetch(ticker instrument.Spec, app App) ([]byte, error) {
	output := []byte{}

	url, err := mkURL(app.Config, ticker.Symbol)
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

func saveData(fpath string, data [][]string) error {
	output := data
	err := osutil.FileExists(fpath)
	if err == nil {
		dataMerged, err := mergeData(fpath, data)
		if err != nil {
			return fmt.Errorf("mergeData %s failed: %v", fpath, err)
		}
		output = dataMerged
	}

	err = writeData(fpath, output)
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

func writeData(fpath string, data [][]string) error {
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
	output := []instrument.Spec{}
	// tmap is unique (key: symbol + security type) instrument collection
	imap := make(map[string]instrument.Spec)

	for _, path := range app.Config.Setup.Watchlists {
		fd, err := os.Open(path)
		if err != nil {
			return output, fmt.Errorf("open %s error: %s", path, err)
		}
		defer fd.Close()

		specLst, err := instrument.SpecLstFromCSV(fd)
		for _, s := range specLst {
			imap[s.Symbol+s.SecurityType.String()] = s
		}
		app.log.Infof("loadTickers: %s - OK", path)
	}

	for _, spec := range imap {
		output = append(output, spec)
	}

	return output, nil
}
