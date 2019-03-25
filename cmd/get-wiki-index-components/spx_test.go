package main

import "testing"

func TestParseSPX(t *testing.T) {
	rows, err := parseSPX()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("ticker: %s, company: %s", r[1], r[0])
	}
}
