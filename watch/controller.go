package watch

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/rsanheim/plur/logger"
)

// Terminals send in-band reports on stdin — focus in/out after a window
// switch, replies to queries a job asked (cursor position, device
// attributes), SGR mouse events — and those bytes arrive glued to whatever
// the user types next. All of them are CSI sequences (ESC [ parameters
// intermediates final), so strip complete CSI sequences before matching
// commands.
var csiSequence = regexp.MustCompile(`\x1b\[[\x30-\x3f]*[\x20-\x2f]*[\x40-\x7e]`)

func stripTerminalReports(line string) string {
	return csiSequence.ReplaceAllString(line, "")
}

// WatcherSource is the watcher lifecycle as the Controller drives it.
// *WatcherManager satisfies it; tests substitute a fake over plain channels
// so no watcher binary is involved.
type WatcherSource interface {
	Start() error
	Stop()
	Events() <-chan Event
	Errors() <-chan error
}

// InterruptibleReader is an input stream whose Close method unblocks any
// active Read. Implementations must allow Read and Close to run concurrently.
type InterruptibleReader interface {
	io.Reader
	io.Closer
}

// ControllerConfig carries everything a watch session needs from the
// process that hosts it. Job selection stays with the caller (the runtime
// package imports watch, so watch cannot select jobs); RunAllJob arrives
// as a ready-to-execute run with no targets — the domain's "run all".
type ControllerConfig struct {
	Planner       Planner
	RunAllJob     JobRun // executed as-is on Enter; empty targets means run all
	DebounceDelay time.Duration
	Timeout       time.Duration // 0 = no timeout
	Watcher       WatcherSource
	Signals       <-chan os.Signal    // caller registers signal.Notify
	Stdin         InterruptibleReader // Controller closes stdin when Run returns
	Stdout        io.Writer
	Stderr        io.Writer
	// OnStarted runs after the watcher starts successfully.
	OnStarted func()
	// Reload replaces the process (terminal reset + exec); it returns only
	// on failure, which ends the session because the watcher has stopped.
	Reload func() error
}

// Controller owns a watch session: the event loop, the REPL, prompt state,
// and the watcher's start/stop. Process-level concerns (signal
// registration, the exec that Reload performs, exit codes) stay with the
// caller.
type Controller struct {
	planner       Planner
	runAllJob     JobRun
	debounceDelay time.Duration
	timeout       time.Duration
	watcher       WatcherSource
	signals       <-chan os.Signal
	stdin         InterruptibleReader
	stdout        io.Writer
	stderr        io.Writer
	onStarted     func()
	reloadFn      func() error

	promptChan chan struct{}
	reloadChan chan struct{}
}

func NewController(cfg ControllerConfig) *Controller {
	return &Controller{
		planner:       cfg.Planner,
		runAllJob:     cfg.RunAllJob,
		debounceDelay: cfg.DebounceDelay,
		timeout:       cfg.Timeout,
		watcher:       cfg.Watcher,
		signals:       cfg.Signals,
		stdin:         cfg.Stdin,
		stdout:        cfg.Stdout,
		stderr:        cfg.Stderr,
		onStarted:     cfg.OnStarted,
		reloadFn:      cfg.Reload,
		promptChan:    make(chan struct{}, 1),
		reloadChan:    make(chan struct{}, 1),
	}
}

// Run blocks for the whole session: it starts the watcher, serves the
// REPL and file events, and returns nil on exit/timeout/signal or an
// error when the watcher fails.
func (c *Controller) Run() error {
	stdinDone := make(chan struct{})
	var stdinWG sync.WaitGroup
	defer func() {
		close(stdinDone)
		_ = c.stdin.Close()
		stdinWG.Wait()
	}()

	if err := c.watcher.Start(); err != nil {
		return err
	}
	defer c.watcher.Stop()
	if c.onStarted != nil {
		c.onStarted()
	}

	debouncer := NewDebouncer(c.debounceDelay)

	var timeoutChan <-chan time.Time
	if c.timeout > 0 {
		timeoutChan = time.After(c.timeout)
	}

	stdinChan := make(chan string, 10)
	stdinWG.Go(func() {
		defer close(stdinChan)
		scanner := bufio.NewScanner(c.stdin)
		for scanner.Scan() {
			input := strings.TrimSpace(stripTerminalReports(scanner.Text()))
			select {
			case stdinChan <- input:
			case <-stdinDone:
				return
			}
		}
	})

	c.printHelp()
	c.showPrompt()

	for {
		select {
		case input, ok := <-stdinChan:
			if !ok {
				stdinChan = nil
				continue
			}
			logger.Logger.Debug("received via stdin", "input", input)
			switch input {
			case "":
				fmt.Fprintln(c.stdout, "Running all tests...")
				if err := ExecuteJob(c.runAllJob, c.planner.CWD); err != nil {
					fmt.Fprintf(c.stderr, "Failed to run: %v\n", err)
				}
				fmt.Fprintln(c.stdout)
				c.showPrompt()
			case "help":
				c.printHelp()
				c.showPrompt()
			case "reload":
				logger.Logger.Debug("User requested process reload")
				c.triggerReload()
			case "debug":
				logger.ToggleDebug()
				if logger.IsDebugEnabled() {
					fmt.Fprintln(c.stdout, "Debug output enabled")
				} else {
					fmt.Fprintln(c.stdout, "Debug output disabled")
				}
				c.showPrompt()
			case "exit":
				fmt.Fprintln(c.stdout, "Exiting watch mode...")
				return nil
			default:
				fmt.Fprintf(c.stdout, "Unknown command: '%s'\n", input)
				c.printHelp()
				c.showPrompt()
			}

		case event := <-c.watcher.Events():
			if event.PathType == "watcher" {
				logger.Logger.Debug("watch", "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType, "associated", fmt.Sprintf("%v", event.Associated))
				continue
			}

			if event.EffectType != "modify" && event.EffectType != "create" {
				continue
			}

			path, ok := c.planner.Admit(event.PathName)
			if !ok {
				continue
			}

			logger.Logger.Debug("watch", "path", path, "fullPath", event.PathName, "event", event.EffectType, "type", event.PathType)

			debouncer.Debounce([]string{path}, func(paths []string) {
				plan := c.planner.Plan(paths)
				for _, run := range plan.Runs {
					if err := ExecuteJob(run, c.planner.CWD); err != nil {
						logger.Logger.Warn("Job execution error", "job", run.Job.Name, "error", err)
					}
				}
				if plan.Reload {
					c.triggerReload()
				}
				if len(plan.Runs) > 0 {
					fmt.Fprintln(c.stdout)
					c.showPrompt()
				}
			})

		case err := <-c.watcher.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", int(c.timeout.Seconds()))
			fmt.Fprintln(c.stdout, "Timeout reached, exiting!")
			return nil

		case sig := <-c.signals:
			switch sig {
			case syscall.SIGINT:
				fmt.Fprintln(c.stdout, "Received SIGINT, shutting down gracefully...")
				return nil
			case syscall.SIGTERM:
				fmt.Fprintln(c.stdout, "Received SIGTERM, shutting down gracefully...")
				return nil
			case syscall.SIGHUP:
				fmt.Fprintln(c.stdout, "Received SIGHUP, reloading plur...")
				if err := c.attemptReload(); err != nil {
					return err
				}
				// Reload execs a new process, so success never reaches here;
				// this return mirrors the original flow for completeness.
				return nil
			default:
				fmt.Fprintf(c.stdout, "Received signal %v, shutting down gracefully...\n", sig)
				return nil
			}

		case <-c.promptChan:
			fmt.Fprint(c.stdout, "[plur] > ")

		case <-c.reloadChan:
			if err := c.attemptReload(); err != nil {
				return err
			}
			// Reload execs a new process, so success never reaches here.
			return nil
		}
	}
}

// attemptReload invokes the process-level reload. It returns only on
// failure — success execs a new process. A failure ends the session because
// the production reload stops the watcher before attempting the exec.
func (c *Controller) attemptReload() error {
	err := c.reloadFn()
	if err != nil {
		logger.Logger.Error("Failed to reload", "error", err)
		fmt.Fprintln(c.stdout, "Failed to reload:", err)
	}
	return err
}

// showPrompt queues a prompt without blocking; a queued prompt is not
// duplicated.
func (c *Controller) showPrompt() {
	select {
	case c.promptChan <- struct{}{}:
	default: // already queued, skip
	}
}

// triggerReload queues a reload without blocking; a queued reload is not
// duplicated.
func (c *Controller) triggerReload() {
	select {
	case c.reloadChan <- struct{}{}:
	default: // already queued, skip
	}
}

func (c *Controller) printHelp() {
	cmdWidth := 20
	fmt.Fprintln(c.stdout, "Available commands")
	fmt.Fprintf(c.stdout, "  %-*s %s\n", cmdWidth, "[Enter]", "Run all tests")
	fmt.Fprintf(c.stdout, "  %-*s %s\n", cmdWidth, "debug", "Toggle debug mode")
	fmt.Fprintf(c.stdout, "  %-*s %s\n", cmdWidth, "help", "Show this help")
	fmt.Fprintf(c.stdout, "  %-*s %s\n", cmdWidth, "reload", "Reload plur")
	fmt.Fprintf(c.stdout, "  %-*s %s\n", cmdWidth, "exit (Ctrl-C)", "Exit watch mode")
	fmt.Fprintln(c.stdout)
}
