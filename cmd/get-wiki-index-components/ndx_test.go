package main

import "testing"

func TestParseNDX(t *testing.T) {
	rows, err := parseNDX()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s", r[1], r[0])
	}
}
