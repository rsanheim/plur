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

// Command builds the ready-to-run command for this job run: argv is
// Job.Cmd plus targets, env is the inherited environment plus Job.Env
// (last entry wins), and Dir is cwd. Execution and display both start
// here so what plur prints is exactly what it runs.
// Job.Cmd must be non-empty; config-load validation and ExecuteJob enforce this.
func (r JobRun) Command(cwd string) *exec.Cmd {
	argv := append(slices.Clone(r.Job.Cmd), r.Targets...)
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = cwd
	cmd.Env = append(os.Environ(), r.Job.Env...)
	return cmd
}

// CommandString renders a command as a shell-style line: the env vars
// plur adds (not the inherited environment), then the args.
func CommandString(cmd *exec.Cmd, addedEnv []string) string {
	parts := append(slices.Clone(addedEnv), cmd.Args...)
	return strings.Join(parts, " ")
}

// ExecuteJob runs a job run from cwd, streaming output to the terminal.
// It brackets the run with a matching pair of log lines: watch fires runs on
// their own goroutines, so the start line alone cannot tell an in-flight run
// from a finished one. Both lines carry the same job and targets at the same
// level so a log reads as a timeline. Failures are logged by the caller.
func ExecuteJob(run JobRun, cwd string) error {
	if len(run.Job.Cmd) == 0 {
		return fmt.Errorf("job %q must define a command", run.Job.Name)
	}
	targets := fmt.Sprintf("%+v", run.Targets)
	logger.Logger.Info("Executing job", "job", run.Job.Name, "targets", targets)

	cmd := run.Command(cwd)
	fmt.Printf("\n[plur] %s\n", CommandString(cmd, run.Job.Env))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	started := time.Now()
	err := cmd.Run()
	logger.Logger.Info("Finished job", "job", run.Job.Name, "targets", targets, "duration", time.Since(started).Round(time.Millisecond))
	return err
}
