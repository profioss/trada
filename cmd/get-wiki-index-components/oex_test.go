package main

import (
	"strings"
	"testing"
)

func TestParseOEX(t *testing.T) {
	components := []struct {
		symbol  string
		company string
	}{
		{
			symbol:  "AAPL",
			company: "Apple",
		},
		{
			symbol:  "AMZN",
			company: "Amazon",
		},
		{
			symbol:  "GOOGL",
			company: "Alphabet",
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

	fname := "testdata/OEX-components.csv.json"
	wd, err := parseWikiData(fname)
	if err != nil {
		t.Fatal(err)
	}
	wdr := strings.NewReader(wd.Parsed.Content.Text)

	rows, err := parseOEX(wdr)
	if err != nil {
		t.Fatal(err)
	}

	minCnt := 99 // minimum number of symbols
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
