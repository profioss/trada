package main

import (
	"strings"
	"testing"
)

func TestParseSPX(t *testing.T) {
	fname := "testdata/SPX-components.csv.json"
	wd, err := parseWikiData(fname)
	if err != nil {
		t.Fatal(err)
	}
	wdr := strings.NewReader(wd.Parsed.Content.Text)

	rows, err := parseSPX(wdr)
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s", r[0], r[1])
	}
}
