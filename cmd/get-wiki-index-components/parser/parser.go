package parser

import (
	"fmt"
	"io"
	"sort"
	"sync"

	"github.com/profioss/trada/model/instrument"
)

var (
	parsersMu sync.RWMutex
	parsers   = make(map[string]Parser)
)

// Parser defines generic parsing behavior.
type Parser interface {
	Parse(io.Reader) ([]instrument.Spec, error)
}

// Register makes a parser available by the provided name.
// If Register is called twice with the same name or if driver is nil,
// it panics.
func Register(name string, parser Parser) {
	parsersMu.Lock()
	defer parsersMu.Unlock()

	if parser == nil {
		panic("parse: Register parser is nil")
	}

	if _, dup := parsers[name]; dup {
		panic("parse: Register called twice for parser " + name)
	}
	parsers[name] = parser
}

// List returns a sorted list of the names of the registered parsers.
func List() []string {
	parsersMu.RLock()
	defer parsersMu.RUnlock()

	var list []string
	for name := range parsers {
		list = append(list, name)
	}
	sort.Strings(list)

	return list
}

// Get returns registered parser by name.
func Get(name string) (Parser, error) {
	parsersMu.RLock()
	p, ok := parsers[name]
	parsersMu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("parse: unknown parser %q (forgotten import?)", name)
	}

	return p, nil
}
