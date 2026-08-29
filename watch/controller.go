package watch

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/rsanheim/plur/logger"
)

// Strip terminal CSI reports that can arrive mixed with user input.
var csiSequence = regexp.MustCompile(`\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]`)

func stripTerminalReports(line string) string {
	return csiSequence.ReplaceAllString(line, "")
}

type WatcherSource interface {
	Start() error
	Stop()
	Events() <-chan Event
	Errors() <-chan error
}

type ControllerConfig struct {
	Planner       Planner
	RunAllJob     JobRun
	DebounceDelay time.Duration
	Timeout       time.Duration
	Watcher       WatcherSource
	Signals       <-chan os.Signal
	Stdin         io.Reader
	StdinIsTTY    bool
	Stdout        io.Writer
	OnStarted     func()
	Reload        func() error
}

type Controller struct {
	cfg        ControllerConfig
	promptChan chan struct{}
}

type runResult struct {
	job *RunningJob
	err error
}

func NewController(cfg ControllerConfig) *Controller {
	return &Controller{
		cfg:        cfg,
		promptChan: make(chan struct{}, 1),
	}
}

func (c *Controller) Run() error {
	if err := c.cfg.Watcher.Start(); err != nil {
		return err
	}
	defer c.cfg.Watcher.Stop()
	if c.cfg.OnStarted != nil {
		c.cfg.OnStarted()
	}

	debouncer := NewDebouncer(c.cfg.DebounceDelay)
	scheduler := NewScheduler()
	batchChan := make(chan TargetSet, 16)
	doneChan := make(chan runResult, 16)
	interruptAlreadyDelivered := false
	var forceAfter time.Duration
	defer func() {
		c.stopRuns(scheduler, doneChan, !interruptAlreadyDelivered, forceAfter)
	}()

	var timeoutChan <-chan time.Time
	if c.cfg.Timeout > 0 {
		timeoutChan = time.After(c.cfg.Timeout)
	}

	stdinChan := make(chan string, 10)
	go func() {
		defer close(stdinChan)
		scanner := bufio.NewScanner(c.cfg.Stdin)
		for scanner.Scan() {
			input := strings.TrimSpace(stripTerminalReports(scanner.Text()))
			stdinChan <- input
		}
	}()

	c.printHelp()
	c.showPrompt()

	for {
		c.drainRunCompletions(scheduler, doneChan)

		select {
		case input, ok := <-stdinChan:
			if !ok {
				stdinChan = nil
				continue
			}
			logger.Logger.Debug("received via stdin", "input", input)
			switch input {
			case "":
				fmt.Fprintln(c.cfg.Stdout, "Running all tests...")
				if !c.startRun(scheduler, doneChan, c.cfg.RunAllJob) {
					c.showPrompt()
				}
			case "help":
				c.printHelp()
				c.showPrompt()
			case "reload":
				logger.Logger.Debug("User requested process reload")
				return c.attemptReload(scheduler, doneChan)
			case "debug":
				logger.ToggleDebug()
				if logger.IsDebugEnabled() {
					fmt.Fprintln(c.cfg.Stdout, "Debug output enabled")
				} else {
					fmt.Fprintln(c.cfg.Stdout, "Debug output disabled")
				}
				c.showPrompt()
			case "exit":
				fmt.Fprintln(c.cfg.Stdout, "Exiting watch mode...")
				return nil
			default:
				fmt.Fprintf(c.cfg.Stdout, "Unknown command: '%s'\n", input)
				c.printHelp()
				c.showPrompt()
			}

		case event := <-c.cfg.Watcher.Events():
			if event.PathType == "watcher" {
				logger.Logger.Debug("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))
				continue
			}

			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			path, ok := c.cfg.Planner.Admit(event.PathName)
			if !ok {
				continue
			}

			logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)

			debouncer.Debounce([]string{path}, func(paths TargetSet) {
				batchChan <- paths
			})

		case paths := <-batchChan:
			c.drainRunCompletions(scheduler, doneChan)
			plan := c.cfg.Planner.Plan(paths)
			started := false
			for _, run := range plan.Runs {
				if c.startRun(scheduler, doneChan, run) {
					started = true
				}
			}
			if !started && len(plan.Runs) > 0 {
				fmt.Fprintln(c.cfg.Stdout)
				c.showPrompt()
			}

		case done := <-doneChan:
			c.finishRun(scheduler, done)

		case err := <-c.cfg.Watcher.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", int(c.cfg.Timeout.Seconds()))
			fmt.Fprintln(c.cfg.Stdout, "Timeout reached, exiting!")
			return nil

		case sig := <-c.cfg.Signals:
			switch sig {
			case syscall.SIGINT:
				if c.cfg.StdinIsTTY {
					interruptAlreadyDelivered = true
					fmt.Fprintln(c.cfg.Stdout)
					fmt.Fprintln(c.cfg.Stdout, "Received SIGINT. Pausing new jobs and waiting for active jobs.")
					fmt.Fprintln(c.cfg.Stdout, "Press Ctrl-C again to terminate.")
				} else {
					forceAfter = 500 * time.Millisecond
					fmt.Fprintln(c.cfg.Stdout, "Received SIGINT, stopping active jobs...")
				}
				return nil
			case syscall.SIGTERM:
				fmt.Fprintln(c.cfg.Stdout, "Received SIGTERM, shutting down gracefully...")
				return nil
			case syscall.SIGHUP:
				fmt.Fprintln(c.cfg.Stdout, "Received SIGHUP, reloading plur...")
				return c.attemptReload(scheduler, doneChan)
			default:
				fmt.Fprintf(c.cfg.Stdout, "Received signal %v, shutting down gracefully...\n", sig)
				return nil
			}

		case <-c.promptChan:
			fmt.Fprint(c.cfg.Stdout, "[plur] > ")
		}
	}
}

func (c *Controller) startRun(scheduler *Scheduler, doneChan chan<- runResult, run JobRun) bool {
	ready, skipped := scheduler.Partition(run)
	c.printSkips(skipped)
	if ready == nil {
		return false
	}

	job, err := StartJob(*ready, c.cfg.Planner.CWD)
	if err != nil {
		c.reportRunError(*ready, err)
		return false
	}
	scheduler.Track(job)

	go func() {
		doneChan <- runResult{
			job: job,
			err: job.Wait(),
		}
	}()
	return true
}

func (c *Controller) finishRun(scheduler *Scheduler, result runResult) {
	scheduler.Release(result.job)
	c.reportRunError(result.job.run, result.err)
	if scheduler.Idle() {
		fmt.Fprintln(c.cfg.Stdout)
		c.showPrompt()
	}
}

func (c *Controller) drainRunCompletions(scheduler *Scheduler, doneChan <-chan runResult) {
	for {
		select {
		case done := <-doneChan:
			c.finishRun(scheduler, done)
		default:
			return
		}
	}
}

func (c *Controller) reportRunError(run JobRun, err error) {
	if err == nil {
		return
	}
	logger.Logger.Warn("Job execution error", "job", run.Job.Name, "targets", run.Targets.Values(), "error", err)
}

func (c *Controller) stopRuns(scheduler *Scheduler, doneChan <-chan runResult, interrupt bool, forceAfter time.Duration) {
	if scheduler.Idle() {
		return
	}

	if interrupt {
		for _, job := range scheduler.RunningJobs() {
			_ = job.Interrupt()
		}
	}

	var forceChan <-chan time.Time
	if forceAfter > 0 {
		forceChan = time.After(forceAfter)
	}
	forceStop := func() {
		for _, job := range scheduler.RunningJobs() {
			_ = job.Kill()
		}
		for !scheduler.Idle() {
			result := <-doneChan
			scheduler.Release(result.job)
		}
	}

	for !scheduler.Idle() {
		select {
		case result := <-doneChan:
			scheduler.Release(result.job)
		case <-forceChan:
			fmt.Fprintln(c.cfg.Stdout, "Shutdown grace period elapsed, forcing active jobs to stop...")
			forceStop()
			return
		case sig := <-c.cfg.Signals:
			if c.cfg.StdinIsTTY {
				fmt.Fprintln(c.cfg.Stdout)
			}
			fmt.Fprintf(c.cfg.Stdout, "Received %v during shutdown, forcing active jobs to stop...\n", sig)
			forceStop()
			return
		}
	}
}

func (c *Controller) printSkips(skipped *JobRun) {
	if skipped == nil {
		return
	}
	if skipped.Targets.Len() == 0 {
		fmt.Fprintf(c.cfg.Stdout, "\n[plur] skipped %s reason=running\n", skipped.Job.Name)
		logger.Logger.Info("Skipped in-flight", "job", skipped.Job.Name, "targets", "[]")
		return
	}

	values := skipped.Targets.Values()
	fmt.Fprintf(c.cfg.Stdout, "\n[plur] skipped %s reason=running\n", strings.Join(values, " "))
	logger.Logger.Info("Skipped in-flight", "job", skipped.Job.Name, "targets", fmt.Sprintf("%+v", values))
}

func (c *Controller) attemptReload(scheduler *Scheduler, doneChan <-chan runResult) error {
	c.stopRuns(scheduler, doneChan, true, 0)
	err := c.cfg.Reload()
	if err != nil {
		logger.Logger.Error("Failed to reload", "error", err)
		fmt.Fprintln(c.cfg.Stdout, "Failed to reload:", err)
	}
	return err
}

func (c *Controller) showPrompt() {
	select {
	case c.promptChan <- struct{}{}:
	default: // already queued, skip
	}
}

func (c *Controller) printHelp() {
	cmdWidth := 20
	fmt.Fprintln(c.cfg.Stdout, "Available commands")
	fmt.Fprintf(c.cfg.Stdout, "  %-*s %s\n", cmdWidth, "[Enter]", "Run all tests")
	fmt.Fprintf(c.cfg.Stdout, "  %-*s %s\n", cmdWidth, "debug", "Toggle debug mode")
	fmt.Fprintf(c.cfg.Stdout, "  %-*s %s\n", cmdWidth, "help", "Show this help")
	fmt.Fprintf(c.cfg.Stdout, "  %-*s %s\n", cmdWidth, "reload", "Reload plur")
	fmt.Fprintf(c.cfg.Stdout, "  %-*s %s\n", cmdWidth, "exit (Ctrl-C)", "Exit watch mode")
	fmt.Fprintln(c.cfg.Stdout)
}
