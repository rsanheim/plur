package watch

// Scheduler tracks the runs currently executing. Controller owns it; callers
// must not use it from job goroutines.
type Scheduler struct {
	nextID   int
	inFlight map[int]JobRun
}

func NewScheduler() *Scheduler {
	return &Scheduler{inFlight: make(map[int]JobRun)}
}

// Claim returns the part of run that may start. Targeted runs drop targets
// already running in the same job. No-targets runs conflict only with another
// no-targets run in the same job.
func (s *Scheduler) Claim(run JobRun) (id int, start *JobRun, skipped []string) {
	if run.NoTargets {
		for _, active := range s.inFlight {
			if active.NoTargets && active.Job.Name == run.Job.Name {
				return 0, nil, nil
			}
		}
		return s.start(run), &run, nil
	}

	free := make([]string, 0, len(run.Targets))
	for _, target := range run.Targets {
		if s.targetInFlight(run.Job.Name, target) {
			skipped = append(skipped, target)
		} else {
			free = append(free, target)
		}
	}
	if len(free) == 0 {
		return 0, nil, skipped
	}

	narrowed := run
	narrowed.Targets = free
	return s.start(narrowed), &narrowed, skipped
}

func (s *Scheduler) Release(id int) {
	delete(s.inFlight, id)
}

func (s *Scheduler) Idle() bool {
	return len(s.inFlight) == 0
}

func (s *Scheduler) start(run JobRun) int {
	s.nextID++
	s.inFlight[s.nextID] = run
	return s.nextID
}

func (s *Scheduler) targetInFlight(jobName, target string) bool {
	for _, active := range s.inFlight {
		if !active.NoTargets && active.Job.Name == jobName {
			for _, activeTarget := range active.Targets {
				if activeTarget == target {
					return true
				}
			}
		}
	}
	return false
}
