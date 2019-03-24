package main

import "testing"

func TestParseDJIA(t *testing.T) {
	rows, err := parseDJIA()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s, exchange: %s", r[2], r[0], r[1])
	}
}
