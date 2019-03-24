package main

import "testing"

func TestParseOEX(t *testing.T) {
	rows, err := parseOEX()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s", r[0], r[1])
	}
}
