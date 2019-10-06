package main

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func parseDJIA(r io.Reader) (output [][]string, err error) {
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
					if link != nil && link.Type == html.ElementNode && link.Data == "a" {
						// extract td element
						row = append(row, extractTextDJIA(thTd))
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
		company := strings.TrimSpace(r[0])
		// exchange := r[1]
		ticker := r[2]
		row := []string{ticker, company}
		output = append(output, row)
	}

	return output, nil
}

func extractTextDJIA(n *html.Node) string {
	output := ""

	link := n.FirstChild
	if link.Type == html.ElementNode && link.Data == "a" {
		output += strings.TrimSpace(link.FirstChild.Data)
	}

	// Symbol column can contain for example: NYSE: MMM
	// detect colon and extract text from following link
	if link.NextSibling != nil && link.NextSibling.Data == ":\u00a0" {
		// link after colon
		link2 := link.NextSibling.NextSibling
		if link2 != nil && link2.Type == html.ElementNode && link2.Data == "a" {
			// return only ticker which is relevant data
			return strings.TrimSpace(link2.FirstChild.Data)
		}
	}

	return output
}
