package main

import (
	"github.com/rsanheim/rux/rspec"
	"github.com/rsanheim/rux/types"
)

// RSpecOutputParser implements TestOutputParser for RSpec JSON output
type RSpecOutputParser struct {
	parser *rspec.OutputParser
}

// NewRSpecOutputParser creates a new RSpec output parser
func NewRSpecOutputParser() *RSpecOutputParser {
	return &RSpecOutputParser{
		parser: &rspec.OutputParser{},
	}
}

// ParseLine parses a single line of RSpec output
func (p *RSpecOutputParser) ParseLine(line string) ([]types.TestNotification, bool) {
	// Now that rspec package uses types package directly, no conversion needed
	return p.parser.ParseLine(line)
}
