package watch

import (
	"testing"

	"github.com/rsanheim/plur/internal/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClaimTargetedRun(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// First targeted run should start with all targets.
	run := JobRun{Job: job, Targets: []string{"spec/a_spec.rb", "spec/b_spec.rb"}}
	id, start, skipped := sched.Claim(run)
	require.NotZero(t, id, "started run should have nonzero id")
	require.NotNil(t, start)
	assert.Equal(t, []string{"spec/a_spec.rb", "spec/b_spec.rb"}, start.Targets)
	assert.Nil(t, skipped)
}

func TestClaimTargetedRunPartialOverlap(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start first run with a and b.
	run1 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb", "spec/b_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// Second run with b and c: b is in flight, c is not.
	run2 := JobRun{Job: job, Targets: []string{"spec/b_spec.rb", "spec/c_spec.rb"}}
	id2, start, skipped := sched.Claim(run2)
	require.NotZero(t, id2, "narrowed run should start")
	assert.Equal(t, []string{"spec/c_spec.rb"}, start.Targets, "only c should remain")
	assert.Equal(t, []string{"spec/b_spec.rb"}, skipped)
}

func TestClaimTargetedRunFullOverlap(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start first run.
	run1 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// Second run with same target: should be fully skipped.
	run2 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id2, start, skipped := sched.Claim(run2)
	assert.Zero(t, id2, "fully overlapping run should not start")
	assert.Nil(t, start)
	assert.Equal(t, []string{"spec/a_spec.rb"}, skipped)
}

func TestClaimNoTargetsRun(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// First no-targets run should start.
	run := JobRun{Job: job, NoTargets: true}
	id, start, skipped := sched.Claim(run)
	require.NotZero(t, id)
	require.NotNil(t, start)
	assert.True(t, start.NoTargets)
	assert.Nil(t, skipped)
}

func TestClaimNoTargetsRunDuplicate(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start first no-targets run.
	run1 := JobRun{Job: job, NoTargets: true}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// Second no-targets run of same job should be skipped.
	run2 := JobRun{Job: job, NoTargets: true}
	id2, start, skipped := sched.Claim(run2)
	assert.Zero(t, id2)
	assert.Nil(t, start)
	assert.Nil(t, skipped, "no-targets skip has no skipped targets")
}

func TestTwoLanesIndependence(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start a targeted run covering a.
	run1 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// A no-targets run should start even though a is in-flight in targeted lane.
	runNT := JobRun{Job: job, NoTargets: true}
	idNT, start, skipped := sched.Claim(runNT)
	require.NotZero(t, idNT)
	assert.NotNil(t, start)
	assert.Nil(t, skipped)

	// A second targeted run covering a should be skipped (a is in targeted in-flight).
	run2 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id2, start2, skipped2 := sched.Claim(run2)
	assert.Zero(t, id2)
	assert.Nil(t, start2)
	assert.Equal(t, []string{"spec/a_spec.rb"}, skipped2)
}

func TestNoTargetsDoesNotBlockTargeted(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start a no-targets run.
	runNT := JobRun{Job: job, NoTargets: true}
	idNT, _, _ := sched.Claim(runNT)
	require.NotZero(t, idNT)

	// A targeted run should start unaffected by the no-targets run.
	run := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id, start, skipped := sched.Claim(run)
	require.NotZero(t, id)
	assert.NotNil(t, start)
	assert.Nil(t, skipped)
}

func TestCrossJobIndependence(t *testing.T) {
	sched := NewScheduler()
	rspec := framework.Job{Name: "rspec"}
	minitest := framework.Job{Name: "minitest"}

	// Start rspec with spec/a_spec.rb.
	run1 := JobRun{Job: rspec, Targets: []string{"spec/a_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// minitest with test/a_test.rb should start; spec/a_spec.rb is rspec's concern.
	run2 := JobRun{Job: minitest, Targets: []string{"test/a_test.rb"}}
	id2, start, skipped := sched.Claim(run2)
	require.NotZero(t, id2)
	assert.NotNil(t, start)
	assert.Nil(t, skipped)

	// rspec with test/a_test.rb should start; that's minitest's in-flight, not rspec's.
	run3 := JobRun{Job: rspec, Targets: []string{"test/a_test.rb"}}
	id3, start3, skipped3 := sched.Claim(run3)
	require.NotZero(t, id3)
	assert.NotNil(t, start3)
	assert.Nil(t, skipped3)
}

func TestRelease(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start a run.
	run := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id, _, _ := sched.Claim(run)
	require.NotZero(t, id)
	assert.False(t, sched.Idle())

	// Same target should be skipped while in flight.
	run2 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id2, start, _ := sched.Claim(run2)
	assert.Zero(t, id2)
	assert.Nil(t, start)

	// Release the first run.
	sched.Release(id)
	assert.True(t, sched.Idle())

	// Now the same target should start.
	id3, start3, skipped3 := sched.Claim(run2)
	require.NotZero(t, id3)
	assert.NotNil(t, start3)
	assert.Nil(t, skipped3)
}

func TestDoubleReleaseSafe(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	run := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id, _, _ := sched.Claim(run)
	require.NotZero(t, id)

	// Double release should not panic.
	sched.Release(id)
	sched.Release(id) // no-op, safe
	assert.True(t, sched.Idle())
}

func TestIdleTracking(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	assert.True(t, sched.Idle())

	// Claim a run.
	run := JobRun{Job: job, Targets: []string{"spec/a_spec.rb"}}
	id, _, _ := sched.Claim(run)
	require.NotZero(t, id)
	assert.False(t, sched.Idle())

	// Release it.
	sched.Release(id)
	assert.True(t, sched.Idle())
}

func TestMultipleConcurrentRuns(t *testing.T) {
	sched := NewScheduler()
	rspec := framework.Job{Name: "rspec"}
	lint := framework.Job{Name: "lint"}

	// Start rspec with spec/a_spec.rb.
	run1 := JobRun{Job: rspec, Targets: []string{"spec/a_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// Start lint with lib/a.rb (different job, no conflict).
	run2 := JobRun{Job: lint, Targets: []string{"lib/a.rb"}}
	id2, _, _ := sched.Claim(run2)
	require.NotZero(t, id2)

	// Both are in flight.
	assert.False(t, sched.Idle())

	// Release first.
	sched.Release(id1)
	assert.False(t, sched.Idle()) // lint still in flight

	// Release second.
	sched.Release(id2)
	assert.True(t, sched.Idle())
}

func TestNarrowingPreservesNewTargets(t *testing.T) {
	sched := NewScheduler()
	job := framework.Job{Name: "rspec"}

	// Start with b and c.
	run1 := JobRun{Job: job, Targets: []string{"spec/b_spec.rb", "spec/c_spec.rb"}}
	id1, _, _ := sched.Claim(run1)
	require.NotZero(t, id1)

	// Claim with a, b, and c: b and c are in flight, a is not.
	run2 := JobRun{Job: job, Targets: []string{"spec/a_spec.rb", "spec/b_spec.rb", "spec/c_spec.rb"}}
	id2, start, skipped := sched.Claim(run2)
	require.NotZero(t, id2)
	require.NotNil(t, start)
	assert.Equal(t, []string{"spec/a_spec.rb"}, start.Targets)
	assert.ElementsMatch(t, []string{"spec/b_spec.rb", "spec/c_spec.rb"}, skipped)
}
