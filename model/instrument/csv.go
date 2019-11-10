package instrument

import (
	"encoding/csv"
	"fmt"
	"io"
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

	return output, nil
}
