package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// OHLCV is one day of historical data.
type OHLCV struct {
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume uint64  `json:"volume"`
}

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

		dataOHLC := []OHLCV{}
		err = json.Unmarshal(data, &dataOHLC)
		if err != nil {
			// store problematic data to .../dir/fname.json.swp for analysis
			fname += ".json.swp"
			app.log.Errorf("%s: unmarshal error: %s; check %s", t, err, fname)
			ioutil.WriteFile(fname, data, filePerms)
			continue
		}
		dataCSV := toCSV(dataOHLC)

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
	url := mkURL(ticker, app.Config)

	resp, err := app.client.Get(url)
	if err != nil {
		return body, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return body, fmt.Errorf("fetch: HTTP Status: %s. URL: %s", resp.Status, url)
	}

	body, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return body, fmt.Errorf("fetch: Read Error: %s", err)
	}

	return body, nil
}

func mkURL(ticker string, conf Config) string {
	return fmt.Sprintf("%s/stock/%s/chart/%s?token=%s",
		conf.Setup.BaseURL, ticker, conf.Setup.Range, conf.Setup.Token)
}

func toCSV(data []OHLCV) [][]string {
	dataCSV := [][]string{}
	// header
	dataCSV = append(dataCSV, []string{"Date", "Open", "High", "Low", "Close", "Volume"})

	for _, ohlc := range data {
		row := []string{
			ohlc.Date,
			fmt.Sprintf("%.2f", ohlc.Open),
			fmt.Sprintf("%.2f", ohlc.High),
			fmt.Sprintf("%.2f", ohlc.Low),
			fmt.Sprintf("%.2f", ohlc.Close),
			fmt.Sprintf("%d", ohlc.Volume),
		}
		dataCSV = append(dataCSV, row)
	}

	return dataCSV
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
		return fmt.Errorf("CSV temp file error: %s", w.Error())
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
