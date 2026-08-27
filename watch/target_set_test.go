package watch

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTargetSetPreservesFirstInsertionOrder(t *testing.T) {
	targets := NewTargetSet("spec/models/user_spec.rb", "spec/lib/user_spec.rb", "spec/models/user_spec.rb")

	assert.Equal(t, []string{"spec/models/user_spec.rb", "spec/lib/user_spec.rb"}, targets.Values())
	assert.Equal(t, 2, targets.Len())
	assert.True(t, targets.Contains("spec/models/user_spec.rb"))
	assert.False(t, targets.Contains("spec/models/missing_spec.rb"))
}

func TestTargetSetValuesCannotMutateSet(t *testing.T) {
	targets := NewTargetSet("spec/models/user_spec.rb")

	values := targets.Values()
	values[0] = "changed"

	assert.Equal(t, []string{"spec/models/user_spec.rb"}, targets.Values())
}

func TestTargetSetOperationsPreserveReceiverOrder(t *testing.T) {
	targets := NewTargetSet("b_spec.rb", "a_spec.rb", "c_spec.rb")
	other := NewTargetSet("c_spec.rb", "b_spec.rb", "d_spec.rb")

	assert.Equal(t, []string{"a_spec.rb"}, targets.Difference(other).Values())
	assert.Equal(t, []string{"b_spec.rb", "c_spec.rb"}, targets.Intersection(other).Values())
}
