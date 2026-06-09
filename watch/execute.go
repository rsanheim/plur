package watch

import (
	"os"
	"os/exec"
	"slices"
	"strings"
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
