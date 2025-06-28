package main

import (
	"github.com/rsanheim/rux/minitest"
	"github.com/rsanheim/rux/rspec"
	"github.com/rsanheim/rux/types"
)

// RSpecParser implements TestOutputParser for RSpec JSON output
type RSpecParser struct {
	parser *rspec.OutputParser
}

// NewRSpecParser creates a new RSpec parser
func NewRSpecParser() *RSpecParser {
	return &RSpecParser{
		parser: &rspec.OutputParser{},
	}
}

// ParseLine parses a single line of RSpec output
func (p *RSpecParser) ParseLine(line string) ([]types.TestNotification, bool) {
	return p.parser.ParseLine(line)
}

func (p *RSpecParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	return p.parser.NotificationToProgress(notification)
}

// MinitestParser implements TestOutputParser for Minitest text output
type MinitestParser struct {
	parser *minitest.OutputParser
}

// NewMinitestParser creates a new Minitest parser
func NewMinitestParser() *MinitestParser {
	return &MinitestParser{
		parser: &minitest.OutputParser{},
	}
}

// ParseLine parses a single line of Minitest output
func (p *MinitestParser) ParseLine(line string) ([]types.TestNotification, bool) {
	return p.parser.ParseLine(line)
}

func (p *MinitestParser) NotificationToProgress(notification types.TestNotification) (string, bool) {
	return p.parser.NotificationToProgress(notification)
}
