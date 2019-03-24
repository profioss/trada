package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"
)

func parseOEX() (output [][]string, err error) {
	// HTML DOM walking can enter unexpected branch which could cause panic
	// e.g. accessing x.FirstChild.NextSibling where FirstChild is nil
	// report panic(s) as error to simplify error handling
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("parseHTML failed: %v", r)
		}
	}()

	fname := "testdata/OEX-components.csv.json"
	wd, err := parseWikiData(fname)
	if err != nil {
		return output, err
	}

	fd, err := os.Create("/tmp/oex.html")
	if err != nil {
		return output, err
	}
	defer fd.Close()
	_, err = io.Copy(fd, strings.NewReader(wd.Parsed.Content.Text))
	if err != nil {
		return output, err
	}

	doc, err := html.Parse(strings.NewReader(wd.Parsed.Content.Text))
	if err != nil {
		return output, err
	}

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			row := []string{}
			for cell := n.FirstChild; cell != nil; cell = cell.NextSibling {
				thTd := cell.NextSibling
				switch {
				case thTd != nil && thTd.Data == "td":
					cell := thTd.FirstChild
					if cell.Type == html.ElementNode && cell.Data == "a" {
						row = append(row, strings.TrimSpace(cell.FirstChild.Data))
					} else {
						row = append(row, strings.TrimSpace(cell.Data))
					}
				case thTd != nil && thTd.Data == "th":
					// process th if needed
				}
			}
			if len(row) >= 2 {
				output = append(output, row)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	return output, nil
}
