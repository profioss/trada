package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// response from API
// this can be either valid data or error info. in both cases HTTP status 200 is returned!

type wikiData struct {
	Parsed jdata `json:"parse"`
}

type jdata struct {
	Title   string  `json:"title"`
	PageID  int64   `json:"pageid"`
	Content content `json:"text"`
}

type content struct {
	Text string `json:"*"`
}

type wikiError struct {
	Error    apiErr `json:"error"`
	ServedBy string `json:"servedby"`
}

type apiErr struct {
	Code string `json:"code"`
	Info string `json:"info"`
	content
}

func parseWikiData(fname string) (wikiData, error) {
	wd := wikiData{}

	fd, err := os.Open(fname)
	if err != nil {
		return wd, err
	}
	defer fd.Close()

	// try parse as wikiError & check if there is actual error message
	wdErr := wikiError{}
	errErr := json.NewDecoder(fd).Decode(&wdErr)
	if errErr == nil && len(wdErr.Error.Code) > 0 {
		return wd, fmt.Errorf("data error: %s, %s", wdErr.Error.Code, wdErr.Error.Info)
	}

	// parse as wikiData if data doesn't contain error details
	_, err = fd.Seek(0, 0)
	if err != nil {
		return wd, fmt.Errorf("rewind file reader failed: %s", err)
	}
	err = json.NewDecoder(fd).Decode(&wd)
	if err != nil {
		return wd, err
	}

	return wd, nil
}
