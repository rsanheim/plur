package main

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rsanheim/rux/logger"
	"github.com/rsanheim/rux/tracing"
	"github.com/rsanheim/rux/types"
)

// streamTestOutput handles the common pattern of streaming test output through a parser
// and collector while sending progress updates to the output channel
func streamTestOutput(
	ctx context.Context,
	stdout, stderr io.Reader,
	parser types.TestOutputParser,
	collector *TestCollector,
	outputChan chan<- OutputMessage,
	workerIndex int,
	testFiles []string,
	framework TestFramework,
	start time.Time,
) (stderrOutput string) {
	var stderrBuilder strings.Builder
	var wg sync.WaitGroup

	// Stream stdout and parse using event-based architecture
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)

		firstOutput := true
		for scanner.Scan() {
			line := scanner.Text()

			if firstOutput {
				// tracing Removed
				firstOutput = false
			}

			// Parse line into notifications
			notifications, consumed := parser.ParseLine(line)

			// If line wasn't consumed by parser, add it as raw output
			if !consumed {
				collector.AddNotification(types.OutputNotification{
					Content: line,
				})
			}
			logger.Logger.Debug("after parseLine", "notifications", notifications)

			// Process each notification
			for _, notification := range notifications {
				// Handle progress events separately - they don't go to collector
				if progress, ok := notification.(types.ProgressEvent); ok {
					if outputChan != nil {
						// Send progress indicator based on character
						switch progress.Character {
						case ".":
							outputChan <- OutputMessage{
								WorkerID: workerIndex,
								Type:     "dot",
							}
						case "F":
							outputChan <- OutputMessage{
								WorkerID: workerIndex,
								Type:     "failure",
							}
						case "E":
							outputChan <- OutputMessage{
								WorkerID: workerIndex,
								Type:     "failure", // Display errors as failures for now
							}
						case "S":
							outputChan <- OutputMessage{
								WorkerID: workerIndex,
								Type:     "pending",
							}
						}
					}
					continue // Don't add progress events to collector
				}

				// Add non-progress notifications to collector
				collector.AddNotification(notification)

				// Only handle suite events for non-progress notifications
				// Test result notifications don't need progress output since we already
				// displayed progress via ProgressEvents
				if outputChan != nil && notification.GetEvent() == types.SuiteStarted {
					if suite, ok := notification.(types.SuiteNotification); ok && suite.LoadTime > 0 {
						tracing.LogEvent(ctx, string(framework)+"_loaded",
							"worker_id", workerIndex,
							"test_files", len(testFiles),
							"load_time", suite.LoadTime.Seconds(),
							"time_since_spawn", time.Since(start).Seconds()*1000)
					}
				}
			}

			// Debug output if enabled
			if os.Getenv("RUX_DEBUG") == "1" && len(notifications) > 0 {
				dump(notifications)
			}
		}

		if err := scanner.Err(); err != nil {
			logger.Logger.Error("error reading stdout", "error", err, "worker", workerIndex)
		}
	}()

	// Stream stderr in real-time
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			stderrBuilder.WriteString("STDERR: " + line + "\n")
			if outputChan != nil {
				outputChan <- OutputMessage{
					WorkerID: workerIndex,
					Type:     "stderr",
					Content:  line,
					Files:    strings.Join(testFiles, ","),
				}
			}
		}
		if err := scanner.Err(); err != nil {
			logger.Logger.Error("error reading stderr", "error", err, "worker", workerIndex)
		}
	}()

	// Wait for all output to be captured
	wg.Wait()

	return stderrBuilder.String()
}
