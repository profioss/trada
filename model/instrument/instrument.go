package instrument

import (
	"fmt"
)

// Spec is Instrument specification.
type Spec struct {
	Symbol       string
	Description  string
	SecurityType Security
	Exchange     string
}

// Validate checks if Spec has valid content.
func (s Spec) Validate() error {
	switch {
	case s.Symbol == "":
		return fmt.Errorf("Symbol not defined")

	case s.SecurityType.Validate() != nil:
		return fmt.Errorf("invalid SecurityType: %v", s.SecurityType.Validate())
	}

	return nil
}
