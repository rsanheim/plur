package main

import (
	"testing"

	"github.com/rsanheim/plur/internal/testruntime"
	"github.com/stretchr/testify/assert"
)

func TestClassifyRunKind(t *testing.T) {
	cases := []struct {
		name            string
		patterns        []string
		tags            []string
		passthroughArgs []string
		aborted         bool
		want            testruntime.RunKind
	}{
		{"default run", nil, nil, nil, false, testruntime.RunKindAggregate},
		{"with patterns no colon", []string{"spec/foo_spec.rb"}, nil, nil, false, testruntime.RunKindAggregate},
		{"file:line pattern", []string{"spec/foo_spec.rb:42"}, nil, nil, false, testruntime.RunKindPartial},
		{"with tag", nil, []string{"focus"}, nil, false, testruntime.RunKindPartial},
		{"with passthrough", nil, nil, []string{"--fail-fast"}, false, testruntime.RunKindPartial},
		{"aborted", nil, nil, nil, true, testruntime.RunKindPartial},
		{"with example id pattern", []string{"spec/foo_spec.rb[1:1]"}, nil, nil, false, testruntime.RunKindPartial},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyRunKind(tc.patterns, tc.tags, tc.passthroughArgs, tc.aborted)
			assert.Equal(t, tc.want, got)
		})
	}
}
