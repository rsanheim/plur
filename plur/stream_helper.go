package main

import (
	"bufio"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/rsanheim/plur/logger"
	"github.com/rsanheim/plur/types"
)

// ScannerBufferSize is the buffer size for scanning test output.
// 256KB allows for large output lines while being memory-efficient.
// Default bufio.Scanner is 64KB which can fail on large single lines.
const ScannerBufferSize = 256 * 1024
const StdErrBufferSize = 1024 * 8

// streamTestOutput handles the common pattern of streaming test output through a parser
// and collector while sending progress updates to the output channel
func streamTestOutput(
	stdout, stderr io.Reader,
	parser types.TestOutputParser,
	collector *TestCollector,
	outputChan chan<- OutputMessage,
	workerIndex int,
	testFiles []string,
	framework TestFramework,
) (stderrOutput string) {
	var stderrBuilder strings.Builder
	stderrBuilder.Grow(StdErrBufferSize) // Pre-allocate for typical stderr output
	var wg sync.WaitGroup

	// Stream stdout and parse using event-based architecture
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		// Increase buffer size to handle large output lines (default is 64KB)
		scanner.Buffer(make([]byte, 0, ScannerBufferSize), ScannerBufferSize)

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
					Event:   types.RawOutput,
					Content: line,
				})
			}
			// Process each notification
			for _, notification := range notifications {

				progressType, isProgress := parser.NotificationToProgress(notification)
				// Handle progress notifications and then continue to next notification
				if isProgress {
					if outputChan != nil {
						outputChan <- OutputMessage{
							WorkerID: workerIndex,
							Type:     progressType,
						}
					}
				}

				// Add all notifications to collector (ProgressEvents will be ignored)
				collector.AddNotification(notification)

			}

			// Debug output if enabled
			if os.Getenv("PLUR_DEBUG") == "1" && len(notifications) > 0 {
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
		// Increase buffer size to handle large output lines (default is 64KB)
		scanner.Buffer(make([]byte, 0, ScannerBufferSize), ScannerBufferSize)
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
