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
	"strings"

	"github.com/profioss/trada/model/ohlc"
	"github.com/profioss/trada/model/ohlc/ohlcio"
	"github.com/profioss/trada/pkg/osutil"
)

func getData(ctx context.Context, app App) error {
	var err error
	tickers := app.Config.symbols // specified by command line param
	if len(tickers) == 0 {        // no symbol specified by command line param
		tickers, err = loadTickers(app)
		if err != nil {
			return fmt.Errorf("loadTickers failed %s", err)
		}
	}
	if len(tickers) < app.Config.Setup.MaxProcs && len(tickers) > 0 {
		app.Config.Setup.MaxProcs = len(tickers)
	}

	for _, t := range tickers {
		data, err := fetch(t, app)
		if err != nil {
			app.log.Errorf("%s: fetch error: %s", t, err)
			continue
		}
		app.log.Debugf("%s: fetch - OK", t)

		fname := filepath.Join(app.Config.Setup.OutputDir, t)

		dataOHLC := []ohlc.OHLC{}
		if app.Config.Setup.Range == "1d" {
			ohlc := ohlc.OHLC{}
			err = json.Unmarshal(data, &ohlc)
			dataOHLC = append(dataOHLC, ohlc)
		} else {
			err = json.Unmarshal(data, &dataOHLC)
		}
		if err != nil {
			// store problematic data to .../dir/fname.json.swp for analysis
			fname += ".json.swp"
			app.log.Errorf("%s: unmarshal error: %s; check %s", t, err, fname)
			osutil.WriteFile(fname, data)
			continue
		}
		dataCSV := ohlcio.ToCSV(dataOHLC)

		fname += ".csv"
		err = saveData(fname, dataCSV)
		if err != nil {
			app.log.Errorf("%s: saveData error: %s", t, err)
		}
		app.log.Debugf("%s: saved to %s", t, fname)
	}

	return nil
}

func fetch(ticker string, app App) ([]byte, error) {
	body := []byte{}
	url, err := mkURL(app.Config, ticker)
	if err != nil {
		return nil, fmt.Errorf("mkUrl failed: %v", err)
	}

	resp, err := app.client.Get(url.String())
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("fetch: HTTP Status: %s. URL: %s", resp.Status, url.String())
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("fetch: Read Error: %s", err)
	}

	return body, nil
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

	toSort := [][]string{}
	for _, record := range data {
		toSort = append(toSort, record)
	}
	sort.Slice(toSort,
		func(i, j int) bool { return toSort[i][0] < toSort[j][0] })

	output = append(output, toSort...)
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

func loadTickers(app App) ([]string, error) {
	// use a map for an easy unique ticker collection
	tmap := make(map[string]string)
	var tickers []string

	for _, path := range app.Config.Setup.Watchlists {
		fd, err := os.Open(path)
		if err != nil {
			return tickers, fmt.Errorf("open %s error: %s", path, err)
		}
		defer fd.Close()

		r := csv.NewReader(fd)
		r.Comma = ';'
		data, err := r.ReadAll()
		if err != nil {
			return tickers, fmt.Errorf("csv read %s error: %s", path, err)
		}

		for i, row := range data {
			if i == 0 {
				continue // skip header
			}
			ticker := strings.TrimSpace(row[0])
			name := strings.TrimSpace(row[1])

			tmap[ticker] = name
		}

		app.log.Infof("loadTickers: %s - OK", path)
	}

	for ticker := range tmap {
		tickers = append(tickers, ticker)
	}

	return tickers, nil
}
