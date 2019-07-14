package ohlcio

import (
	"fmt"

	"github.com/profioss/trada/model/ohlc"
)

// CSVheader defines CSV column header.
var CSVheader = []string{"Date", "Open", "High", "Low", "Close", "Volume"}

// ToCSV converts []ohlc.OHLC into CSV representation which is [][]string.
func ToCSV(ohlcLst []ohlc.OHLC) [][]string {
	dataCSV := [][]string{}
	dataCSV = append(dataCSV, CSVheader)

	for _, ohlc := range ohlcLst {
		row := []string{
			ohlc.Date.String(),
			fmt.Sprintf("%.2f", ohlc.Open),
			fmt.Sprintf("%.2f", ohlc.High),
			fmt.Sprintf("%.2f", ohlc.Low),
			fmt.Sprintf("%.2f", ohlc.Close),
			fmt.Sprintf("%d", ohlc.Volume),
		}
		dataCSV = append(dataCSV, row)
	}

	return dataCSV
}
