package wiki

import (
	"encoding/json"
	"fmt"
	"io"
)

// response from API
// this can be either valid data or error info. in both cases HTTP status 200 is returned!

// Data is parsed  Wiki API data.
type Data struct {
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

// Error is Wiki API error response.
type Error struct {
	Error    APIerr `json:"error"`
	ServedBy string `json:"servedby"`
}

// APIerr is Wiki API error content.
type APIerr struct {
	Code string `json:"code"`
	Info string `json:"info"`
	content
}

// Parse parses input as error or Data.
func Parse(r io.ReadSeeker) (Data, error) {
	// func parseWikiData(fname string) (Data, error) {
	d := Data{}

	// try parse as wikiError & check if there is actual error message
	dErr := Error{}
	err := json.NewDecoder(r).Decode(&dErr)
	if err == nil && len(dErr.Error.Code) > 0 {
		return d, fmt.Errorf("data error: %s, %s", dErr.Error.Code, dErr.Error.Info)
	}

	// parse as wikiData if data doesn't contain error details
	_, err = r.Seek(0, 0)
	if err != nil {
		return d, fmt.Errorf("rewind IO reader failed: %s", err)
	}
	err = json.NewDecoder(r).Decode(&d)
	if err != nil {
		return d, err
	}

	return d, nil
}
