package watch

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"time"

	"github.com/rsanheim/plur/logger"
)

// Execution and display share this path so Plur prints exactly what it runs.
// Job.Env wins over inherited variables. Job.Cmd must be non-empty.
func (r JobRun) Command(cwd string) *exec.Cmd {
	argv := append(slices.Clone(r.Job.Cmd), r.Targets.Values()...)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), r.Job.Env...)
	return cmd
}

// Inherited environment variables are intentionally omitted from display.
func CommandString(cmd *exec.Cmd, addedEnv []string) string {
	parts := append(slices.Clone(addedEnv), cmd.Args...)
	return strings.Join(parts, " ")
}

// The goroutine calling Wait owns process reaping.
type RunningJob struct {
	cmd     *exec.Cmd
	run     JobRun
	started time.Time
}

func StartJob(run JobRun, cwd string) (*RunningJob, error) {
	if len(run.Job.Cmd) == 0 {
		return nil, fmt.Errorf("job %q must define a command", run.Job.Name)
	}

	cmd := run.Command(cwd)
	fmt.Printf("\n[plur] %s\n", CommandString(cmd, run.Job.Env))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	logger.Logger.Info("Executing job", "job", run.Job.Name, "targets", fmt.Sprintf("%+v", run.Targets.Values()))
	return &RunningJob{cmd: cmd, run: run, started: time.Now()}, nil
}

func (j *RunningJob) Wait() error {
	err := j.cmd.Wait()
	logger.Logger.Info("Finished job", "job", j.run.Job.Name, "targets", fmt.Sprintf("%+v", j.run.Targets.Values()), "duration", time.Since(j.started).Round(time.Millisecond))
	return err
}

func (j *RunningJob) Interrupt() error {
	return j.cmd.Process.Signal(os.Interrupt)
}

func (j *RunningJob) Kill() error {
	return j.cmd.Process.Kill()
}
