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
	"time"

	"github.com/profioss/trada/model/ohlc"
	"github.com/profioss/trada/model/ohlc/ohlcio"
	"github.com/profioss/trada/pkg/osutil"
	"github.com/profioss/trada/pkg/typedef"
)

// respCW is Cryptowatch representation of OHLC data.
type respCW struct {
	Result    map[string][][]json.Number `json:"result"`
	Allowance allowance                  `json:"allowance"`
}

type allowance struct {
	Cost      int64 `json:"cost"`
	Remaining int64 `json:"remaining"`
}

func parse(data []byte) ([]ohlc.OHLC, error) {
	output := []ohlc.OHLC{}
	resp := respCW{}

	err := json.Unmarshal(data, &resp)
	if err != nil {
		return output, err
	}

	if len(resp.Result) != 1 {
		return output,
			fmt.Errorf("parse: expected map with 1 key equal to 86400, got %d key(s)", len(resp.Result))
	}
	input, ok := resp.Result["86400"]
	if !ok {
		return output,
			fmt.Errorf("parse: expected map with 1 key equal to 86400, the key not found")
	}

	output = make([]ohlc.OHLC, 0, len(input))
	for _, bar := range input {
		ohlc, err := ohlcFromCWbar(bar)
		if err != nil {
			return output, err
		}
		output = append(output, ohlc)
	}

	return output, nil
}

func ohlcFromCWbar(data []json.Number) (ohlc.OHLC, error) {
	output := ohlc.OHLC{}

	if len(data) < 7 {
		return output, fmt.Errorf("unexpected CW response: wanted 7 elements; data: %v", data)
	}

	ts, err := data[0].Int64()
	if err != nil {
		return output, fmt.Errorf("invalid timestamp %q; data: %v", data[0].String(), data)
	}
	// CW's UNIX timestamp means end of given bar.
	// As for daily bars it is bit tricky
	// e.g. 2019-10-29 daily bar has end at 2019-10-30 00:00:00 eg NEXT DAY!
	// In this case we want 2019-10-29 bar with date 2019-10-29 :)
	// that's why we need to go back 1 day.
	output.Date = typedef.Date(time.Unix(ts, 0).UTC().AddDate(0, 0, -1))

	o, err := data[1].Float64()
	if err != nil {
		return output, fmt.Errorf("invalid open %q; data: %v", data[1].String(), data)
	}
	output.Open = o

	h, err := data[2].Float64()
	if err != nil {
		return output, fmt.Errorf("invalid high %q; data: %v", data[2].String(), data)
	}
	output.High = h

	l, err := data[3].Float64()
	if err != nil {
		return output, fmt.Errorf("invalid low %q; data: %v", data[3].String(), data)
	}
	output.Low = l

	c, err := data[4].Float64()
	if err != nil {
		return output, fmt.Errorf("invalid close %q; data: %v", data[4].String(), data)
	}
	output.Close = c

	// we want VolumeBase (index 5) not VolumeQuote (index 6)
	// see type Interval: https://github.com/cryptowatch/cw-sdk-go/blob/master/common/markets.go
	v, err := data[5].Float64()
	if err != nil {
		return output, fmt.Errorf("invalid volume %q; data: %v", data[5].String(), data)
	}
	// TODO uint64 is not suitable for crypto
	output.Volume = uint64(v)

	return output, nil
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

		dataOHLC, err := parse(data)
		if err != nil {
			// store problematic data to .../dir/fname.json.swp for analysis
			fname += ".json.swp"
			app.log.Errorf("%s: parse error: %s; check %s", t, err, fname)
			osutil.WriteFile(fname, data)
			continue
		}
		app.log.Debugf("%s: parse - OK", t)

		fname += ".csv"
		dataCSV := ohlcio.ToCSV(dataOHLC)
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

// mkURL generates proper API URL
// see https://cryptowat.ch/docs/api#market-ohlc
func mkURL(conf Config, ticker string) (url.URL, error) {
	str := fmt.Sprintf("%s/markets/%s/%s/ohlc",
		conf.Setup.BaseURL, conf.Setup.Exchange, ticker)

	u, err := url.Parse(str)
	if err != nil {
		return url.URL{}, fmt.Errorf("invalid URL string: %v", err)
	}

	after, err := conf.Setup.Range.since()
	if err != nil {
		return url.URL{}, fmt.Errorf("range %q error: %v", conf.Setup.Range, err)
	}
	before := time.Now().UTC().Truncate(time.Hour * 24) // today at 00:00:00

	q := u.Query()
	q.Set("after", fmt.Sprintf("%d", after.Unix()))
	q.Set("before", fmt.Sprintf("%d", before.Unix()))
	q.Set("periods", "86400") // 1 day aka daily timeframe

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
