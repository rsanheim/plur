package watch

// Controller owns this state; job goroutines must not access it.
type scheduler struct {
	inFlight map[*runningJob]struct{}
}

func newScheduler() *scheduler {
	return &scheduler{inFlight: make(map[*runningJob]struct{})}
}

// Targeted runs drop targets already running in the same job. Bare runs
// conflict only with another bare run in the same job.
func (s *scheduler) partition(run JobRun) (ready, skipped *JobRun) {
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

func (s *scheduler) track(job *runningJob) {
	s.inFlight[job] = struct{}{}
}

func (s *scheduler) release(job *runningJob) {
	delete(s.inFlight, job)
}

func (s *scheduler) runningJobs() []*runningJob {
	jobs := make([]*runningJob, 0, len(s.inFlight))
	for job := range s.inFlight {
		jobs = append(jobs, job)
	}
	return jobs
}

func (s *scheduler) idle() bool {
	return len(s.inFlight) == 0
}

func (s *scheduler) targetsInFlight(jobName string) TargetSet {
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
