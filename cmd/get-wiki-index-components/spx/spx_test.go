package spx

import (
	"strings"
	"testing"
)

func TestParseSPX(t *testing.T) {
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

	minCnt := 499 // minimum number of symbols
	if len(rows) < minCnt {
		t.Fatalf("expected at least %d parsed symbols, got: %d", minCnt, len(rows))
	}

	found := []string{}
	for _, c := range components {
		for _, r := range rows {
			if r.Symbol == c.symbol && strings.Contains(r.Description, c.company) {
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
