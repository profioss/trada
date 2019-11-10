package instrument

import (
	"fmt"
	"strings"
)

// Security is a tradable financial asset
// https://en.wikipedia.org/wiki/Security_(finance)
type Security int

const (
	// Invalid means Security type was not set.
	Invalid Security = iota

	// Equity - common stocks, equity indices, ETFs, etc.
	Equity

	// Forex - foreign currencies.
	Forex

	// Crypto - cryptocurrencies, digital assets
	Crypto

	// Bond - debt securities.
	// Bond

	// Future - futures contracts.
	// Future

	// Option - options contracts.
	// Option
)

// see SecurityDecimalPlaces function.
var secDecimalPlacesMap = map[Security]int{
	Equity: 2,
	Forex:  4,
	Crypto: 8,
}

var secStrMap = map[Security]string{
	Invalid: "invalid",
	Equity:  "equity",
	Forex:   "forex",
	Crypto:  "crypto",
}

// Validate checks if Security is valid.
func (s Security) Validate() error {
	for sec := range secStrMap {
		if sec == s {
			if sec == Invalid {
				return fmt.Errorf("Security not set")
			}
			return nil
		}
	}

	return fmt.Errorf("unknown Security: %d", s)
}

func (s Security) String() string {
	str, ok := secStrMap[s]
	if !ok {
		return ""
	}
	return str
}

// SecurityDecimalPlaces returns decimal places of given security.
// For example equities are quoted in cents (2 decimal places),
// cryptocurrencies are quoted in satoshis (8 decimal places).
func SecurityDecimalPlaces(s Security) int {
	d, ok := secDecimalPlacesMap[s]
	if !ok {
		return 0
	}
	return d
}

// SecurityFromString parses input string and returns Security.
func SecurityFromString(s string) (Security, error) {
	sx := strings.ToLower(strings.TrimSpace(s))

	for sec, str := range secStrMap {
		if str == sx {
			return sec, sec.Validate()
		}
	}

	return Invalid, fmt.Errorf("invalid security string: %q", s)
}
