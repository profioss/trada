package typedef

import (
	"fmt"
	"time"
)

// DateFormat specifies date formatting - YYYY-MM-DD.
const DateFormat = "2006-01-02"

// Date represents time as date.
type Date time.Time

// DateFromStr creates new Date from string in format YYYY-MM-DD.
func DateFromStr(s string) (Date, error) {
	t, err := time.Parse(DateFormat, s)
	d := Date(t)
	return d, err
}

// String formats Date as YYYY-MM-DD.
func (d Date) String() string {
	return d.Time().Format(DateFormat)
}

// Time provides access to time.Time type which is underlying type for Date
func (d Date) Time() time.Time {
	return time.Time(d)
}

//
// JSON serialization
//

// UnmarshalJSON - JSON unmarshaller of Date
func (d *Date) UnmarshalJSON(b []byte) (err error) {
	if b[0] == '"' && b[len(b)-1] == '"' {
		b = b[1 : len(b)-1]
	}
	*d, err = DateFromStr(string(b))
	return
}

// MarshalJSON - JSON marshaller of Date
func (d *Date) MarshalJSON() ([]byte, error) {
	format := fmt.Sprintf("%q", DateFormat)
	return []byte(d.Time().Format(format)), nil
}
