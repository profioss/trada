package instrument

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

// SpecLstToCSV exports []Spec to CSV.
func SpecLstToCSV(w io.Writer, ss []Spec) error {
	data := make([][]string, 0, len(ss))
	// CSV output header
	data = append(data, []string{"sym", "name", "security"})
	for _, s := range ss {
		data = append(data,
			[]string{s.Symbol, s.Description, s.SecurityType.String()})
	}

	wcsv := csv.NewWriter(w)
	wcsv.Comma = ';'
	wcsv.WriteAll(data)
	if wcsv.Error() != nil {
		return fmt.Errorf("CSV write error: %v", wcsv.Error())
	}

	return nil
}

// SpecLstFromCSV imports []Spec from CSV.
// NOTE this function expect data exported by complementary
// SpecLstToCSV function.
func SpecLstFromCSV(r io.Reader) ([]Spec, error) {
	output := []Spec{}

	rcsv := csv.NewReader(r)
	rcsv.Comma = ';'
	data, err := rcsv.ReadAll()
	if err != nil {
		return output, fmt.Errorf("CSV read error: %v", err)
	}

	for _, row := range data[1:] { // skip header
		spec := Spec{
			Symbol:      strings.TrimSpace(row[0]),
			Description: strings.TrimSpace(row[1]),
		}

		sec, err := SecurityFromString(row[2])
		if err != nil {
			return output, err
		}
		spec.SecurityType = sec
	}

	return output, nil
}
