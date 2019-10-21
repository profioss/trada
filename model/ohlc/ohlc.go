package ohlc

import (
	"sort"

	"github.com/profioss/trada/pkg/typedef"
)

// OHLC represents Open High Low Close values of market data.
type OHLC struct {
	Date   typedef.Date `json:"date"`
	Open   float64      `json:"open"`
	High   float64      `json:"high"`
	Low    float64      `json:"low"`
	Close  float64      `json:"close"`
	Volume uint64       `json:"volume"`
}

// Validate checks correctness of OHLC data.
func (o *OHLC) Validate() error {
	return nil
}

// Vec is vector of OHLC data with some handy getter/setter functions.
type Vec struct {
	dates []typedef.Date
	data  map[typedef.Date]OHLC
}

// Dates provides sorted list of Dates in OHLC vector.
func (v *Vec) Dates() []typedef.Date {
	output := []typedef.Date{}
	return output
}

// GetByDate returns OHLC by date or error if not found in the vector.
func (v *Vec) GetByDate(d typedef.Date) (OHLC, error) {
	return OHLC{}, nil
}

// GetByIdx returns OHLC by index or error if not found in the vector.
func (v *Vec) GetByIdx(i int) (OHLC, error) {
	return OHLC{}, nil
}

// SetByDate returns OHLC by date or error if not found in the vector.
func (v *Vec) SetByDate(d typedef.Date) error {
	return nil
}

// Validate checks correctness of Vec data.
func (v *Vec) Validate() error {
	return nil
}

// NewVec creates Vec.
func NewVec(lst []OHLC) (Vec, error) {
	v := Vec{}

	// unique data
	data := make(map[typedef.Date]OHLC, len(lst))
	for _, ohlc := range lst {
		data[ohlc.Date] = ohlc
	}
	v.data = data

	// sorted unique date list
	dates := make([]typedef.Date, 0, len(lst))
	for d := range data {
		dates = append(dates, d)
	}
	sort.Slice(dates,
		func(i, j int) bool {
			return dates[i].Time().Unix() < dates[j].Time().Unix()
		})
	v.dates = dates

	return v, v.Validate()
}
