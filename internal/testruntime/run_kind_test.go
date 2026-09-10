package testruntime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRunKind_IsAggregateEligible(t *testing.T) {
	assert.True(t, RunKindAggregate.IsAggregateEligible())
	assert.False(t, RunKindPartial.IsAggregateEligible())
}

func TestClassifyRunKind(t *testing.T) {
	cases := []struct {
		name            string
		patterns        []string
		tags            []string
		passthroughArgs []string
		want            RunKind
	}{
		{"default run", nil, nil, nil, RunKindAggregate},
		{"with patterns no colon", []string{"spec/foo_spec.rb"}, nil, nil, RunKindAggregate},
		{"file:line pattern", []string{"spec/foo_spec.rb:42"}, nil, nil, RunKindPartial},
		{"with tag", nil, []string{"focus"}, nil, RunKindPartial},
		{"with passthrough", nil, nil, []string{"--fail-fast"}, RunKindPartial},
		{"with example id pattern", []string{"spec/foo_spec.rb[1:1]"}, nil, nil, RunKindPartial},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ClassifyRunKind(tc.patterns, tc.tags, tc.passthroughArgs)
			assert.Equal(t, tc.want, got)
		})
	}
}
