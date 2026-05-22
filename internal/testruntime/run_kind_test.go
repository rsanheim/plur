package testruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyRunKind(t *testing.T) {
	cases := []struct {
		name            string
		patterns        []string
		tags            []string
		passthroughArgs []string
		aborted         bool
		want            RunKind
	}{
		{"default run", nil, nil, nil, false, RunKindAggregate},
		{"with patterns no colon", []string{"spec/foo_spec.rb"}, nil, nil, false, RunKindAggregate},
		{"file:line pattern", []string{"spec/foo_spec.rb:42"}, nil, nil, false, RunKindPartial},
		{"with tag", nil, []string{"focus"}, nil, false, RunKindPartial},
		{"with passthrough", nil, nil, []string{"--fail-fast"}, false, RunKindPartial},
		{"aborted", nil, nil, nil, true, RunKindPartial},
		{"with example id pattern", []string{"spec/foo_spec.rb[1:1]"}, nil, nil, false, RunKindPartial},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRunKind(tc.patterns, tc.tags, tc.passthroughArgs, tc.aborted)
			assert.Equal(t, tc.want, got)
		})
	}
}
