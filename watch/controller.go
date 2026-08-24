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
// *WatcherManager satisfies it.
type WatcherSource interface {
	Start() error
	Stop()
	Events() <-chan Event
	Errors() <-chan error
}

// ControllerConfig carries everything a watch session needs from the
// process that hosts it.
type ControllerConfig struct {
	Planner       Planner
	RunAllJob     JobRun // executed as-is on Enter; empty targets means run all
	DebounceDelay time.Duration
	Timeout       time.Duration // 0 = no timeout
	Watcher       WatcherSource
	Signals       <-chan os.Signal // caller registers signal.Notify
	Stdin         io.Reader
	Stdout        io.Writer
	Stderr        io.Writer
	// Reload replaces the process; it returns only on failure, and the
	// session keeps running when it does.
	Reload func() error
}

// Controller owns a watch session: the event loop, the REPL, prompt state,
// and the watcher's start/stop. Process-level concerns (signal
// registration, the exec behind Reload, exit codes) stay with the caller.
type Controller struct {
	cfg        ControllerConfig
	promptChan chan struct{}
	reloadChan chan struct{}
}

func NewController(cfg ControllerConfig) *Controller {
	return &Controller{
		cfg:        cfg,
		promptChan: make(chan struct{}, 1),
		reloadChan: make(chan struct{}, 1),
	}
}

// Run blocks for the whole session: nil on exit/timeout/signal, an error
// when the watcher fails.
func (c *Controller) Run() error {
	if err := c.cfg.Watcher.Start(); err != nil {
		return err
	}
	defer c.cfg.Watcher.Stop()

	debouncer := NewDebouncer(c.cfg.DebounceDelay)

	var timeoutChan <-chan time.Time
	if c.cfg.Timeout > 0 {
		timeoutChan = time.After(c.cfg.Timeout)
	}

	stdinChan := make(chan string, 10)
	go func() {
		scanner := bufio.NewScanner(c.cfg.Stdin)
		for scanner.Scan() {
			stdinChan <- strings.TrimSpace(stripTerminalReports(scanner.Text()))
		}
	}()

	c.printHelp()
	c.showPrompt()

	for {
		select {
		case input := <-stdinChan:
			logger.Logger.Debug("received via stdin", "input", input)
			switch input {
			case "":
				fmt.Fprintln(c.cfg.Stdout, "Running all tests...")
				if err := ExecuteJob(c.cfg.RunAllJob, c.cfg.Planner.CWD); err != nil {
					fmt.Fprintf(c.cfg.Stderr, "Failed to run: %v\n", err)
				}
				fmt.Fprintln(c.cfg.Stdout)
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

			debouncer.Debounce([]string{path}, func(paths []string) {
				plan := c.cfg.Planner.Plan(paths)
				for _, run := range plan.Runs {
					if err := ExecuteJob(run, c.cfg.Planner.CWD); err != nil {
						logger.Logger.Warn("Job execution error", "job", run.Job.Name, "error", err)
					}
				}
				if plan.Reload {
					c.triggerReload()
				}
				if len(plan.Runs) > 0 {
					fmt.Fprintln(c.cfg.Stdout)
					c.showPrompt()
				}
			})

		case err := <-c.cfg.Watcher.Errors():
			return fmt.Errorf("watcher error: %v", err)

		case <-timeoutChan:
			logger.Logger.Info("plur timeout reached, exiting!", "event", "timeout", "timeout", int(c.cfg.Timeout.Seconds()))
			fmt.Fprintln(c.cfg.Stdout, "Timeout reached, exiting!")
			return nil

		case sig := <-c.cfg.Signals:
			switch sig {
			case syscall.SIGINT:
				fmt.Fprintln(c.cfg.Stdout, "Received SIGINT, shutting down gracefully...")
				return nil
			case syscall.SIGTERM:
				fmt.Fprintln(c.cfg.Stdout, "Received SIGTERM, shutting down gracefully...")
				return nil
			case syscall.SIGHUP:
				fmt.Fprintln(c.cfg.Stdout, "Received SIGHUP, reloading plur...")
				if err := c.attemptReload(); err != nil {
					continue
				}
				// Reload execs a new process on success, so this is reached
				// only in tests with a fake Reload.
				return nil
			default:
				fmt.Fprintf(c.cfg.Stdout, "Received signal %v, shutting down gracefully...\n", sig)
				return nil
			}

		case <-c.promptChan:
			fmt.Fprint(c.cfg.Stdout, "[plur] > ")

		case <-c.reloadChan:
			c.attemptReload()
		}
	}
}

// attemptReload returns only on failure — success execs a new process —
// and the session keeps running: the error is reported and the prompt
// comes back.
func (c *Controller) attemptReload() error {
	err := c.cfg.Reload()
	if err != nil {
		logger.Logger.Error("Failed to reload", "error", err)
		fmt.Fprintln(c.cfg.Stdout, "Failed to reload:", err)
		c.showPrompt()
	}
	return err
}

func (c *Controller) showPrompt() {
	select {
	case c.promptChan <- struct{}{}:
	default: // already queued, skip
	}
}

func (c *Controller) triggerReload() {
	select {
	case c.reloadChan <- struct{}{}:
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
