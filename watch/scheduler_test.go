package watch

import (
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScheduler_TargetPartitioning(t *testing.T) {
	scheduler := newScheduler()
	rspec := framework.Job{Name: "rspec"}

	first, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb", "b_spec.rb")})
	require.NotNil(t, first)
	assert.Nil(t, skipped)
	firstJob := &runningJob{run: *first}
	scheduler.track(firstJob)

	second, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("b_spec.rb", "c_spec.rb")})
	require.NotNil(t, second)
	require.NotNil(t, skipped)
	assert.Equal(t, []string{"c_spec.rb"}, second.Targets.Values())
	assert.Equal(t, rspec, second.Job)
	assert.Equal(t, []string{"b_spec.rb"}, skipped.Targets.Values())
	assert.Equal(t, rspec, skipped.Job)
	secondJob := &runningJob{run: *second}
	scheduler.track(secondJob)

	duplicateSecondRun, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("c_spec.rb")})
	assert.Nil(t, duplicateSecondRun)
	require.NotNil(t, skipped)
	assert.Equal(t, []string{"c_spec.rb"}, skipped.Targets.Values())

	duplicate, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb")})
	assert.Nil(t, duplicate)
	require.NotNil(t, skipped)
	assert.Equal(t, []string{"a_spec.rb"}, skipped.Targets.Values())

	scheduler.release(firstJob)
	afterRelease, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("a_spec.rb")})
	require.NotNil(t, afterRelease)
	assert.Nil(t, skipped)
}

func TestScheduler_IndependentJobsAndBareLane(t *testing.T) {
	scheduler := newScheduler()
	rspec := framework.Job{Name: "rspec"}
	rubocop := framework.Job{Name: "rubocop"}

	targeted, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("user.rb")})
	require.NotNil(t, targeted)
	assert.Nil(t, skipped)
	scheduler.track(&runningJob{run: *targeted})

	otherJob, skipped := scheduler.partition(JobRun{Job: rubocop, Targets: NewTargetSet("user.rb")})
	require.NotNil(t, otherJob)
	assert.Nil(t, skipped)
	scheduler.track(&runningJob{run: *otherJob})

	bare, skipped := scheduler.partition(JobRun{Job: rspec})
	require.NotNil(t, bare)
	assert.Nil(t, skipped)
	scheduler.track(&runningJob{run: *bare})

	duplicate, skipped := scheduler.partition(JobRun{Job: rspec})
	assert.Nil(t, duplicate)
	require.NotNil(t, skipped)
	assert.Equal(t, rspec, skipped.Job)
	assert.Empty(t, skipped.Targets.Values())

	besideBare, skipped := scheduler.partition(JobRun{Job: rspec, Targets: NewTargetSet("account.rb")})
	require.NotNil(t, besideBare)
	assert.Nil(t, skipped)
}
