package main

import (
	"strings"
	"testing"
)

func TestParseDJIA(t *testing.T) {
	components := []struct {
		symbol  string
		company string
	}{
		{
			symbol:  "AAPL",
			company: "Apple",
		},
		{
			symbol:  "JNJ",
			company: "Johnson & Johnson",
		},
		{
			symbol:  "JPM",
			company: "JPMorgan Chase",
		},
		{
			symbol:  "MSFT",
			company: "Microsoft",
		},
	}

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

	minCnt := 29 // minimum number of symbols
	if len(rows) < minCnt {
		t.Fatalf("expected at least %d parsed symbols, got: %d", minCnt, len(rows))
	}

	found := []string{}
	for _, c := range components {
		for _, r := range rows {
			if r[0] == c.symbol && strings.Contains(r[1], c.company) {
				found = append(found, c.symbol)
				break
			}
		}
	}

	if len(components) != len(found) {
		t.Errorf("Of %d components %d found: %s",
			len(components), len(found), strings.Join(found, ", "))
	}
}
