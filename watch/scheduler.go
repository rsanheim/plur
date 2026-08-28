package watch

// Scheduler tracks the runs currently executing. Controller owns it; callers
// must not use it from job goroutines.
type Scheduler struct {
	nextID   int
	inFlight map[int]scheduledRun
}

type scheduledRun struct {
	jobName string
	targets TargetSet
}

func NewScheduler() *Scheduler {
	return &Scheduler{inFlight: make(map[int]scheduledRun)}
}

// Claim returns the part of run that may start. Targeted runs drop targets
// already running in the same job. Bare runs conflict only with another bare
// run in the same job.
func (s *Scheduler) Claim(run JobRun) (id int, start *JobRun, skipped TargetSet) {
	if run.Targets.Len() == 0 {
		for _, active := range s.inFlight {
			if active.targets.Len() == 0 && active.jobName == run.Job.Name {
				return 0, nil, skipped
			}
		}
		return s.start(run), &run, skipped
	}

	inFlight := s.targetsInFlight(run.Job.Name)
	free := run.Targets.Difference(inFlight)
	skipped = run.Targets.Intersection(inFlight)
	if free.Len() == 0 {
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
	s.inFlight[s.nextID] = scheduledRun{
		jobName: run.Job.Name,
		targets: run.Targets,
	}
	return s.nextID
}

func (s *Scheduler) targetsInFlight(jobName string) TargetSet {
	var targets []string
	for _, active := range s.inFlight {
		if active.jobName == jobName {
			for target := range active.targets.All() {
				targets = append(targets, target)
			}
		}
	}
	return NewTargetSet(targets...)
}
