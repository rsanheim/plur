package main

import (
	"bufio"
	"io"
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

// ANSI color codes
const (
	colorGreen  = "\033[32m"
	colorRed    = "\033[31m"
	colorYellow = "\033[33m"
	colorReset  = "\033[0m"
)

// Pre-compiled output strings to avoid repeated concatenation
var (
	greenDot   = []byte(colorGreen + "." + colorReset)
	redF       = []byte(colorRed + "F" + colorReset)
	redE       = []byte(colorRed + "E" + colorReset)
	yellowStar = []byte(colorYellow + "*" + colorReset)
	plainDot   = []byte(".")
	plainF     = []byte("F")
	plainE     = []byte("E")
	plainStar  = []byte("*")
)

func streamTestOutput(
	stdout, stderr io.Reader,
	parser types.TestOutputParser,
	collector *TestCollector,
	outputChan chan<- OutputMessage,
	workerIndex int,
	streamStdout bool, // Only stream unconsumed stdout for RSpec (JSON makes it safe)
) (stderrOutput string) {
	var stderrBuilder strings.Builder
	stderrBuilder.Grow(StdErrBufferSize) // Pre-allocate for typical stderr output
	var wg sync.WaitGroup

	// Stream stdout and parse using event-based architecture
	// Uses bufio.Reader instead of Scanner to avoid hanging on lines > 256KB
	// (Scanner has a hard token limit that causes ErrTooLong and stops draining)
	wg.Go(func() {
		reader := bufio.NewReaderSize(stdout, ScannerBufferSize)

		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				logger.Logger.Error("error reading stdout", "error", err, "worker", workerIndex)
				break
			}

			// Trim trailing newline/carriage return
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			// Handle EOF: process any remaining content, then exit
			if err == io.EOF {
				if line == "" {
					break
				}
				// Process final line without newline, then exit after this iteration
			}

			notifications, consumed := parser.ParseLine(line)

			for _, notification := range notifications {
				progressType, isProgress := parser.NotificationToProgress(notification)
				// Handle progress notifications
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

			// If line wasn't consumed by parser, add it as raw output
			if !consumed {
				collector.AddNotification(types.OutputNotification{
					Event:   types.RawOutput,
					Content: line,
				})
				// Stream stdout in real-time for RSpec (JSON formatter makes this safe)
				// Don't stream for Minitest - it returns consumed=false for everything,
				// so streaming would duplicate all output
				if outputChan != nil && streamStdout {
					outputChan <- OutputMessage{
						WorkerID:    workerIndex,
						Type:        "stdout",
						Content:     line,
						CurrentFile: parser.CurrentFile(),
					}
				}
			}

			if err == io.EOF {
				break
			}
		}
	})

	// Stream stderr in real-time
	// Uses bufio.Reader instead of Scanner to avoid hanging on lines > 256KB
	wg.Go(func() {
		reader := bufio.NewReaderSize(stderr, ScannerBufferSize)

		for {
			line, err := reader.ReadString('\n')
			if err != nil && err != io.EOF {
				logger.Logger.Error("error reading stderr", "error", err, "worker", workerIndex)
				break
			}

			// Trim trailing newline/carriage return
			line = strings.TrimSuffix(line, "\n")
			line = strings.TrimSuffix(line, "\r")

			// Handle EOF: process any remaining content, then exit
			if err == io.EOF {
				if line == "" {
					break
				}
			}

			stderrBuilder.WriteString(line)
			stderrBuilder.WriteString("\n")
			if outputChan != nil {
				outputChan <- OutputMessage{
					WorkerID: workerIndex,
					Type:     "stderr",
					Content:  line,
				}
			}

			if err == io.EOF {
				break
			}
		}
	})

	// Wait for all output to be captured
	wg.Wait()

	return stderrBuilder.String()
}
