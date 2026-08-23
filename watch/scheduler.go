package watch

// Scheduler tracks which JobRuns are executing so duplicate work can be
// skipped. Not goroutine-safe by design: it is owned by the watch event
// loop and must only be touched from it.
type Scheduler struct {
	nextID   int
	inFlight map[int]JobRun // the runs actually executing, by run id
}

func NewScheduler() *Scheduler {
	return &Scheduler{inFlight: make(map[int]JobRun)}
}

// Claim decides what of a run may start, per the two lanes:
//
//	no-targets lane: run.NoTargets — starts unless the same job already
//	               has a no-targets run in flight (whole-run skip); target
//	               overlap is never consulted, in either direction.
//	targeted lane: each target held by a targeted in-flight run of the
//	               same job is dropped and reported in skipped; the run
//	               starts narrowed to the rest, or not at all.
//
// A started run gets a fresh id (always > 0), is stored until Release(id).
// A fully-suppressed run returns id 0 and start nil: nothing was stored,
// so there is nothing to release. A suppressed no-targets run has skipped nil —
// the caller prints the job-level skip line from run.Job.Name.
func (s *Scheduler) Claim(run JobRun) (id int, start *JobRun, skipped []string) {
	if run.NoTargets {
		if s.noTargetsInFlight(run.Job.Name) {
			return 0, nil, nil
		}
		return s.start(run), &run, nil
	}
	var free []string
	for _, t := range run.Targets {
		if s.targetInFlight(run.Job.Name, t) { // scans targeted runs only
			skipped = append(skipped, t)
		} else {
			free = append(free, t)
		}
	}
	if len(free) == 0 {
		return 0, nil, skipped
	}
	narrowed := run
	narrowed.Targets = free
	return s.start(narrowed), &narrowed, skipped
}

// Release removes a finished run from the in-flight set.
func (s *Scheduler) Release(id int) { delete(s.inFlight, id) }

// Idle reports whether nothing is in flight (drives the prompt).
func (s *Scheduler) Idle() bool { return len(s.inFlight) == 0 }

// start assigns a fresh id and stores the run.
func (s *Scheduler) start(run JobRun) int {
	s.nextID++
	s.inFlight[s.nextID] = run
	return s.nextID
}

// noTargetsInFlight reports whether a no-targets run of jobName is in flight.
func (s *Scheduler) noTargetsInFlight(jobName string) bool {
	for _, r := range s.inFlight {
		if r.Job.Name == jobName && r.NoTargets {
			return true
		}
	}
	return false
}

// targetInFlight reports whether a targeted run of jobName covers target.
// Ignores no-targets runs per the two-lane rules.
func (s *Scheduler) targetInFlight(jobName string, target string) bool {
	for _, r := range s.inFlight {
		if r.Job.Name == jobName && !r.NoTargets {
			for _, t := range r.Targets {
				if t == target {
					return true
				}
			}
		}
	}
	return false
}
