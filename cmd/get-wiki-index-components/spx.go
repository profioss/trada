package main

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func parseSPX(r io.Reader) (output [][]string, err error) {
	// HTML DOM walking can enter unexpected branch which could cause panic
	// e.g. accessing x.FirstChild.NextSibling where FirstChild is nil
	// report panic(s) as error to simplify error handling
	defer func() {
		if rec := recover(); rec != nil {
			err = fmt.Errorf("parseHTML failed: %v", rec)
		}
	}()

	doc, err := html.Parse(r)
	if err != nil {
		return output, err
	}

	rows := [][]string{}
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "tr" {
			row := []string{}
			for cell := n.FirstChild; cell != nil; cell = cell.NextSibling {
				thTd := cell.NextSibling
				switch {
				case thTd != nil && thTd.Data == "td":
					link := thTd.FirstChild
					if link == nil {
						continue
					}
					if link.Type == html.ElementNode && link.Data == "a" {
						row = append(row, strings.TrimSpace(link.FirstChild.Data))
					}
				case thTd != nil && thTd.Data == "th":
					// process th if needed
				}
			}
			if len(row) >= 3 {
				rows = append(rows, row)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	for _, r := range rows {
		company := r[0]
		ticker := r[1]
		row := []string{ticker, company}
		output = append(output, row)
	}

	return output, nil
}
