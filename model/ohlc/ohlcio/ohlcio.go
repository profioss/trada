package ohlcio

import (
	"fmt"

	"github.com/profioss/trada/model/instrument"
	"github.com/profioss/trada/model/ohlc"
)

// CSVheader defines CSV column header.
var CSVheader = []string{"Date", "Open", "High", "Low", "Close", "Volume"}

// ToCSV converts []ohlc.OHLC into CSV representation which is [][]string.
func ToCSV(ohlcLst []ohlc.OHLC, s instrument.Security) [][]string {
	dataCSV := [][]string{}
	dataCSV = append(dataCSV, CSVheader)

	decimalPlaces := int32(instrument.SecurityDecimalPlaces(s))

	for _, ohlc := range ohlcLst {
		row := []string{
			ohlc.Date.String(),
			fmt.Sprintf("%s", ohlc.Open.StringFixed(decimalPlaces)),
			fmt.Sprintf("%s", ohlc.High.StringFixed(decimalPlaces)),
			fmt.Sprintf("%s", ohlc.Low.StringFixed(decimalPlaces)),
			fmt.Sprintf("%s", ohlc.Close.StringFixed(decimalPlaces)),
		}
		switch s {
		case instrument.Crypto: // volume with decimal places
			row = append(row, fmt.Sprintf("%s", ohlc.Volume.StringFixed(decimalPlaces)))
		default: // volume as integer
			row = append(row, fmt.Sprintf("%s", ohlc.Volume.StringFixed(0)))
		}

		dataCSV = append(dataCSV, row)
	}

	return dataCSV
}
