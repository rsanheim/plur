package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldHintWatchSharedHelper(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{name: "RSpec helper", path: "spec/spec_helper.rb", expected: true},
		{name: "RSpec support file", path: "spec/support/factory_bot.rb", expected: true},
		{name: "RSpec test file", path: "spec/models/user_spec.rb", expected: false},
		{name: "Minitest helper", path: "test/test_helper.rb", expected: true},
		{name: "Minitest test file", path: "test/models/user_test.rb", expected: false},
		{name: "Non-Ruby file", path: "spec/README.md", expected: false},
		{name: "Unrelated Ruby file", path: "lib/calculator.rb", expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, shouldHintWatchSharedHelper(tt.path))
		})
	}
}
