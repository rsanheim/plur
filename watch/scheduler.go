package watch

// Scheduler tracks the runs currently executing. Controller owns it; callers
// must not use it from job goroutines.
type Scheduler struct {
	inFlight map[*RunningJob]struct{}
}

func NewScheduler() *Scheduler {
	return &Scheduler{inFlight: make(map[*RunningJob]struct{})}
}

// Partition returns the part of run that may start. Targeted runs drop targets
// already running in the same job. Bare runs conflict only with another bare
// run in the same job.
func (s *Scheduler) Partition(run JobRun) (ready, skipped *JobRun) {
	if run.Targets.Len() == 0 {
		for active := range s.inFlight {
			if active.run.Targets.Len() == 0 && active.run.Job.Name == run.Job.Name {
				skipped := run
				return nil, &skipped
			}
		}
		return &run, nil
	}

	inFlight := s.targetsInFlight(run.Job.Name)
	free := run.Targets.Difference(inFlight)
	blocked := run.Targets.Intersection(inFlight)
	if free.Len() > 0 {
		readyRun := run
		readyRun.Targets = free
		ready = &readyRun
	}
	if blocked.Len() > 0 {
		skippedRun := run
		skippedRun.Targets = blocked
		skipped = &skippedRun
	}
	return ready, skipped
}

func (s *Scheduler) Track(job *RunningJob) {
	s.inFlight[job] = struct{}{}
}

func (s *Scheduler) Release(job *RunningJob) {
	delete(s.inFlight, job)
}

func (s *Scheduler) RunningJobs() []*RunningJob {
	jobs := make([]*RunningJob, 0, len(s.inFlight))
	for job := range s.inFlight {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *Scheduler) Idle() bool {
	return len(s.inFlight) == 0
}

func (s *Scheduler) targetsInFlight(jobName string) TargetSet {
	var targets []string
	for active := range s.inFlight {
		if active.run.Job.Name == jobName {
			for target := range active.run.Targets.All() {
				targets = append(targets, target)
			}
		}
	}
	return NewTargetSet(targets...)
}
