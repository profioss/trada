package equity

import (
	"fmt"

	"github.com/profioss/trada/model/instrument"
	"github.com/profioss/trada/model/ohlc"
)

// Equity represents equity financial instrument.
type Equity struct {
	Spec instrument.Spec
	Vec  ohlc.Vec
}

// Validate checks if Spec has valid content.
func (e Equity) Validate() error {
	switch {
	case e.Spec.Validate() != nil:
		return fmt.Errorf("invalid Spec: %v", e.Spec.Validate())

	case e.Vec.Validate() != nil:
		return fmt.Errorf("invalid Vec: %v", e.Vec.Validate())
	}

	return nil
}
