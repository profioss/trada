package ndx

import (
	"fmt"
	"io"
	"strings"

	"github.com/profioss/trada/cmd/get-wiki-index-components/parser"
	"github.com/profioss/trada/model/instrument"

	"golang.org/x/net/html"
)

// Parser is wiki parser for DJIA components.
type Parser struct{}

func init() {
	parser.Register("NDX", &Parser{})
}

// Parse parses wiki API data and returns list of instrument.Spec.
func (p *Parser) Parse(r io.Reader) (output []instrument.Spec, err error) {
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
				rows = append(rows, row)
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}
	f(doc)

	for _, r := range rows {
		iSpec := instrument.Spec{
			Symbol:       strings.TrimSpace(r[1]),
			Description:  strings.TrimSpace(r[0]),
			SecurityType: instrument.Equity,
			// Exchange:     strings.TrimSpace(r[x]), // not available at the time
		}
		output = append(output, iSpec)
	}

	return output, nil
}
