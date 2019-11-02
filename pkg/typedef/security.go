package typedef

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

// SecurityDecimalPlaces returns decimal places of given security.
// For example equities are quoted in cents (2 decimal places),
// cryptocurrencies are quoted in satoshis (8 decimal places).
func SecurityDecimalPlaces(s Security) int {
	switch s {
	case Equity:
		return 2
	case Forex:
		return 4
	case Crypto:
		return 8
	default:
		return 0
	}
}
