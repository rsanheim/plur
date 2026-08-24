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

type ControllerConfig struct {
	Planner       Planner
	RunAllJob     JobRun
	DebounceDelay time.Duration
	Timeout       time.Duration
	Watcher       WatcherSource
	Signals       <-chan os.Signal
	Stdin         InterruptibleReader
	Stdout        io.Writer
	Stderr        io.Writer
	OnStarted     func()
	Reload        func() error
}

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

func (c *Controller) Run() error {
	stdinDone := make(chan struct{})
	var stdinWG sync.WaitGroup
	defer func() {
		close(stdinDone)
		_ = c.cfg.Stdin.Close()
		stdinWG.Wait()
	}()

	if err := c.cfg.Watcher.Start(); err != nil {
		return err
	}
	defer c.cfg.Watcher.Stop()
	if c.cfg.OnStarted != nil {
		c.cfg.OnStarted()
	}

	debouncer := NewDebouncer(c.cfg.DebounceDelay)

	var timeoutChan <-chan time.Time
	if c.cfg.Timeout > 0 {
		timeoutChan = time.After(c.cfg.Timeout)
	}

	stdinChan := make(chan string, 10)
	stdinWG.Go(func() {
		defer close(stdinChan)
		scanner := bufio.NewScanner(c.cfg.Stdin)
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
				return c.attemptReload()
			default:
				fmt.Fprintf(c.cfg.Stdout, "Received signal %v, shutting down gracefully...\n", sig)
				return nil
			}

		case <-c.promptChan:
			fmt.Fprint(c.cfg.Stdout, "[plur] > ")

		case <-c.reloadChan:
			return c.attemptReload()
		}
	}
}

func (c *Controller) attemptReload() error {
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
