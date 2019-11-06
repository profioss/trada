package ohlc

import (
	"fmt"
	"sort"
	"time"

	"github.com/profioss/trada/pkg/typedef"

	"github.com/shopspring/decimal"
)

// OHLC represents Open High Low Close values of market data.
// This is for timeframe >= 1 day. For intraday data use OHLCintrad.
type OHLC struct {
	Date   typedef.Date    `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
}

// Validate checks correctness of OHLC data.
func (o *OHLC) Validate() error {
	switch {
	case o.Date.Time().Before(time.Now().AddDate(-300, 0, 0)):
		return fmt.Errorf("Date: %s is too far away", o.Date.String())

	case o.Open.IsNegative():
		return fmt.Errorf("Open: %s is less than zero", o.Open)

	case o.High.IsNegative():
		return fmt.Errorf("High: %s is less than zero", o.High)

	case o.High.LessThan(o.Low):
		return fmt.Errorf("High: %s is less than Low: %s", o.High, o.Low)

	case o.Low.IsNegative():
		return fmt.Errorf("Low: %s is less than zero", o.Low)

	case o.Close.IsNegative():
		return fmt.Errorf("Close: %s is less than zero", o.Close)

	case o.Volume.IsNegative():
		return fmt.Errorf("Volume: %s is less than zero", o.Volume)
	}
	return nil
}

// OHLCintrad represents Open High Low Close values of market data.
// This is for timeframe >= 1 day. For intraday data use OHLCintrad.
type OHLCintrad struct {
	Date   typedef.Date    `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume decimal.Decimal `json:"volume"`
}

// Validate checks correctness of OHLC data.
func (oid *OHLCintrad) Validate() error {
	switch {
	case oid.Date.Time().Before(time.Now().AddDate(-300, 0, 0)):
		return fmt.Errorf("Date: %s is too far away", oid.Date.String())

	case oid.Open.IsNegative():
		return fmt.Errorf("Open: %s is less than zero", oid.Open)

	case oid.High.IsNegative():
		return fmt.Errorf("High: %s is less than zero", oid.High)

	case oid.High.LessThan(oid.Low):
		return fmt.Errorf("High: %s is less than Low: %s", oid.High, oid.Low)

	case oid.Low.IsNegative():
		return fmt.Errorf("Low: %s is less than zero", oid.Low)

	case oid.Close.IsNegative():
		return fmt.Errorf("Close: %s is less than zero", oid.Close)

	case oid.Volume.IsNegative():
		return fmt.Errorf("Volume: %s is less than zero", oid.Volume)
	}
	return nil
}

// Vec represents vector of OHLC data with some handy getter functions.
// NOTE: Vec is designed to be immutable. It is not designed to be fast.
// Each getter method returns new copy of requested data
// i.e. NEW DATA ALLOCATION FOR EACH METHOD CALL!
// If you use data multiple times, store it to variable and reuse
// e.g. dates := v.Dates() // reuse dates
type Vec struct {
	dates     []typedef.Date
	data      map[typedef.Date]OHLC
	timeframe time.Duration
	maxGap    time.Duration
}

// Data provides sorted (by Date) list of OHLC elements.
func (v *Vec) Data() []OHLC {
	output := make([]OHLC, 0, len(v.data))

	for _, d := range v.dates {
		bar, ok := v.data[d]
		if !ok {
			panic(fmt.Sprintf("internal inconsistency: missing data for %s", d))
		}
		output = append(output, bar)
	}

	return output
}

// Dates provides sorted list of Dates in OHLC vector.
func (v *Vec) Dates() []typedef.Date {
	output := make([]typedef.Date, 0, len(v.dates))

	for _, d := range v.dates {
		output = append(output, d)
	}

	return output
}

// At returns OHLC by date or error if not found.
func (v *Vec) At(d typedef.Date) (OHLC, error) {
	output := OHLC{}

	o, ok := v.data[d]
	if !ok {
		return output, fmt.Errorf("data for date %s not found", d)
	}
	output = o

	return output, nil
}

// AtIdx returns OHLC by index or error if not found.
func (v *Vec) AtIdx(i int) (OHLC, error) {
	output := OHLC{}

	if i < 0 || i >= len(v.dates) {
		return output, fmt.Errorf("invalid index: %d, no data", i)
	}

	d := v.dates[i]
	output, ok := v.data[d]
	if !ok {
		return output, fmt.Errorf("internal inconsistency: missing data for %s at index %d", d, i)
	}

	return output, nil
}

// Validate checks correctness of Vec data.
func (v *Vec) Validate() error {
	for _, bar := range v.Data() {
		if bar.Validate() != nil {
			return fmt.Errorf("%s - %s", bar.Date, bar.Validate())
		}
	}

	return nil
}

// NewVec creates Vec.
func NewVec(lst []OHLC, timeframe time.Duration) (Vec, error) {
	v := Vec{timeframe: timeframe}

	// unique data
	data := make(map[typedef.Date]OHLC, len(lst))
	for _, ohlc := range lst {
		data[ohlc.Date] = ohlc
	}
	v.data = data

	// sorted unique date list
	dates := make([]typedef.Date, 0, len(data))
	for d := range data {
		dates = append(dates, d)
	}
	sort.Slice(dates,
		func(i, j int) bool {
			return dates[i].Time().Unix() < dates[j].Time().Unix()
		})
	v.dates = dates

	var maxGap time.Duration
	for i := range dates { // ensure sorted dates
		if i == 0 {
			continue
		}

		d := dates[i].Time()
		dPrev := dates[i-1].Time()
		gap := d.Sub(dPrev)
		if gap > maxGap {
			maxGap = gap
		}
	}
	v.maxGap = maxGap

	return v, v.Validate()
}
