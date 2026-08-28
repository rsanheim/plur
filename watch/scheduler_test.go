package watch

import (
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_TargetClaims(t *testing.T) {
	scheduler := NewScheduler()
	rspec := framework.Job{Name: "rspec"}

	firstID, first, skipped := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb", "b_spec.rb")})
	require.NotNil(t, first)
	assert.Positive(t, firstID)
	assert.Empty(t, skipped)

	secondID, second, skipped := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("b_spec.rb", "c_spec.rb")})
	require.NotNil(t, second)
	assert.Positive(t, secondID)
	assert.Equal(t, []string{"c_spec.rb"}, second.Targets.Values())
	assert.Equal(t, []string{"b_spec.rb"}, skipped.Values())

	_, duplicateSecondRun, skipped := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("c_spec.rb")})
	assert.Nil(t, duplicateSecondRun)
	assert.Equal(t, []string{"c_spec.rb"}, skipped.Values())

	_, duplicate, skipped := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb")})
	assert.Nil(t, duplicate)
	assert.Equal(t, []string{"a_spec.rb"}, skipped.Values())

	scheduler.Release(firstID)
	_, afterRelease, skipped := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb")})
	require.NotNil(t, afterRelease)
	assert.Empty(t, skipped)
}

func TestScheduler_IndependentJobsAndBareLane(t *testing.T) {
	scheduler := NewScheduler()
	rspec := framework.Job{Name: "rspec"}
	rubocop := framework.Job{Name: "rubocop"}

	_, targeted, _ := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("user.rb")})
	require.NotNil(t, targeted)

	_, otherJob, _ := scheduler.Claim(JobRun{Job: rubocop, Targets: NewTargetSet("user.rb")})
	require.NotNil(t, otherJob)

	bareID, bare, _ := scheduler.Claim(JobRun{Job: rspec})
	require.NotNil(t, bare)
	assert.Positive(t, bareID)

	_, duplicate, skipped := scheduler.Claim(JobRun{Job: rspec})
	assert.Nil(t, duplicate)
	assert.Empty(t, skipped)

	_, besideBare, _ := scheduler.Claim(JobRun{Job: rspec, Targets: NewTargetSet("account.rb")})
	require.NotNil(t, besideBare)
}
