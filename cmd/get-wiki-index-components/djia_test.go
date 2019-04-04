package main

import (
	"strings"
	"testing"
)

func TestParseDJIA(t *testing.T) {
	fname := "testdata/DJIA-components.csv.json"
	wd, err := parseWikiData(fname)
	if err != nil {
		t.Fatal(err)
	}
	wdr := strings.NewReader(wd.Parsed.Content.Text)

	rows, err := parseDJIA(wdr)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s", r[0], r[1])
	}
}
