package watch

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rsanheim/plur/logger"
)

// Watch mode runs one job at a time. runMu serializes execution so a burst of
// saves queues instead of starting concurrent suites against the same test
// database, and current holds the in-flight process so shutdown can stop it.
var (
	runMu   sync.Mutex
	procMu  sync.Mutex
	current *os.Process
)

// terminateGrace is how long a job gets to exit after SIGTERM before SIGKILL.
const terminateGrace = 2 * time.Second

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
func ExecuteJob(run JobRun, cwd string) error {
	if len(run.Job.Cmd) == 0 {
		return fmt.Errorf("job %q must define a command", run.Job.Name)
	}
	runMu.Lock()
	defer runMu.Unlock()

	logger.Logger.Info("Executing job", "job", run.Job.Name, "targets", fmt.Sprintf("%+v", run.Targets))

	cmd := run.Command(cwd)
	fmt.Printf("\n[plur] %s\n", CommandString(cmd, run.Job.Env))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		return err
	}
	setCurrent(cmd.Process)
	defer setCurrent(nil)
	return cmd.Wait()
}

func setCurrent(p *os.Process) {
	procMu.Lock()
	current = p
	procMu.Unlock()
}

func currentProcess() *os.Process {
	procMu.Lock()
	defer procMu.Unlock()
	return current
}

// TerminateRunningJob stops an in-flight job so it does not outlive plur.
// Ruby test runners rescue Interrupt to print a summary, so the terminal's
// Ctrl-C does not reliably stop them and plur must signal explicitly.
// Grandchildren of a shell-wrapper job are not reached; see docs.
func TerminateRunningJob() {
	p := currentProcess()
	if p == nil {
		return
	}
	logger.Logger.Debug("Terminating in-flight job", "pid", p.Pid)
	_ = p.Signal(syscall.SIGTERM)

	deadline := time.Now().Add(terminateGrace)
	for time.Now().Before(deadline) {
		if currentProcess() != p {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	_ = p.Kill()
}
