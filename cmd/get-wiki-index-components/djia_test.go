package main

import "testing"

func TestParseDJIA(t *testing.T) {
	rows, err := parseDJIA()
	if err != nil {
		t.Fatal(err)
	}

	for _, r := range rows {
		t.Logf("%s : %s : %s", r[0], r[1], r[2])
	}
}
