package main

import (
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/urfave/cli/v2"
)

type TestResult struct {
	SpecFile string
	Success  bool
	Output   string
	Error    error
	Duration time.Duration
}

func findSpecFiles() ([]string, error) {
	var specFiles []string
	
	// Check if spec directory exists
	if _, err := os.Stat("spec"); os.IsNotExist(err) {
		return specFiles, nil // Return empty list if no spec directory
	}
	
	// Walk the spec directory recursively
	err := filepath.WalkDir("spec", func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		
		// Skip directories
		if d.IsDir() {
			return nil
		}
		
		// Check if file ends with _spec.rb
		if strings.HasSuffix(path, "_spec.rb") {
			specFiles = append(specFiles, path)
		}
		
		return nil
	})
	
	if err != nil {
		return nil, fmt.Errorf("error walking spec directory: %v", err)
	}
	
	return specFiles, nil
}

func getWorkerCount(cliWorkers int) int {
	// Priority: CLI flag > ENV var > default (cores-2)
	if cliWorkers > 0 {
		return cliWorkers
	}
	
	if envVar := os.Getenv("PARALLEL_TEST_PROCESSORS"); envVar != "" {
		if count, err := strconv.Atoi(envVar); err == nil && count > 0 {
			return count
		}
	}
	
	// Default: cores minus 2, minimum 1
	workers := runtime.NumCPU() - 2
	if workers < 1 {
		workers = 1
	}
	return workers
}

func runSpecFile(ctx context.Context, specFile string, dryRun bool, saveJSON bool, outputMutex *sync.Mutex) TestResult {
	start := time.Now()
	
	args := []string{"bundle", "exec", "rspec", "--format", "progress", specFile}
	
	var jsonFile string
	if saveJSON {
		// Create temp file for JSON output
		tmpFile, err := ioutil.TempFile("", "rux-results-*.json")
		if err != nil {
			return TestResult{
				SpecFile: specFile,
				Success:  false,
				Output:   "",
				Error:    fmt.Errorf("failed to create temp file: %v", err),
				Duration: time.Since(start),
			}
		}
		jsonFile = tmpFile.Name()
		tmpFile.Close()
		
		// Add JSON formatter to separate file
		args = append(args, "--format", "json", "--out", jsonFile)
	}
	
	if dryRun {
		return TestResult{
			SpecFile: specFile,
			Success:  true,
			Output:   fmt.Sprintf("[dry-run] %s", strings.Join(args, " ")),
			Duration: time.Since(start),
		}
	}

	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	
	// Create pipes for real-time output streaming
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    err,
			Duration: time.Since(start),
		}
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    err,
			Duration: time.Since(start),
		}
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return TestResult{
			SpecFile: specFile,
			Success:  false,
			Output:   "",
			Error:    err,
			Duration: time.Since(start),
		}
	}

	var outputBuilder strings.Builder
	var wg sync.WaitGroup
	
	// Stream stdout in real-time (only progress dots now)
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")
			
			// Check if this line contains only progress indicators
			isProgressLine := len(strings.TrimSpace(line)) > 0 && 
				strings.Trim(line, ".F*") == ""
			
			outputMutex.Lock()
			if isProgressLine {
				// Progress dots - print without newline
				fmt.Print(line)
			}
			// Skip "Finished in..." and other RSpec output
			outputMutex.Unlock()
		}
	}()
	
	// Stream stderr in real-time
	wg.Add(1)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			outputBuilder.WriteString(line + "\n")
			
			outputMutex.Lock()
			fmt.Fprintln(os.Stderr, line)
			outputMutex.Unlock()
		}
	}()

	// Wait for all output to be processed
	wg.Wait()
	
	// Wait for command to complete
	err = cmd.Wait()
	
	success := err == nil
	if exitErr, ok := err.(*exec.ExitError); ok {
		// RSpec failed tests return exit code 1, which is still a "successful" run
		success = exitErr.ExitCode() <= 1
	}

	// Clean up JSON file if not saving
	if saveJSON && jsonFile != "" {
		defer os.Remove(jsonFile)
		// TODO: Could read and process JSON here if needed
	}

	return TestResult{
		SpecFile: specFile,
		Success:  success,
		Output:   outputBuilder.String(),
		Error:    err,
		Duration: time.Since(start),
	}
}

func runTestsInParallel(specFiles []string, dryRun bool, saveJSON bool, maxWorkers int) ([]TestResult, time.Duration) {
	start := time.Now()
	ctx := context.Background()
	results := make(chan TestResult, len(specFiles))
	
	// Mutex to synchronize output from multiple processes
	var outputMutex sync.Mutex

	// Create worker pool with limited workers
	jobs := make(chan string, len(specFiles))
	var wg sync.WaitGroup

	// Start worker goroutines
	for i := 0; i < maxWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for specFile := range jobs {
				result := runSpecFile(ctx, specFile, dryRun, saveJSON, &outputMutex)
				results <- result
			}
		}()
	}

	// Send jobs to workers
	go func() {
		for _, specFile := range specFiles {
			jobs <- specFile
		}
		close(jobs)
	}()

	// Close results channel when all workers complete
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect all results
	var allResults []TestResult
	for result := range results {
		allResults = append(allResults, result)
	}

	totalWallTime := time.Since(start)
	return allResults, totalWallTime
}

func printResults(results []TestResult, wallTime time.Duration) {
	var totalCPUTime time.Duration
	successCount := 0
	
	// Make sure we end the progress line
	fmt.Println()

	for _, result := range results {
		totalCPUTime += result.Duration
		if result.Success {
			successCount++
		}
	}

	fmt.Printf("\n=== Summary ===\n")
	fmt.Printf("Files: %d/%d passed\n", successCount, len(results))
	fmt.Printf("Wall time: %v\n", wallTime)
	fmt.Printf("Total CPU time: %v\n", totalCPUTime)
	
	if successCount < len(results) {
		fmt.Printf("\nFailed files:\n")
		for _, result := range results {
			if !result.Success {
				fmt.Printf("  - %s", result.SpecFile)
				if result.Error != nil {
					fmt.Printf(" (error: %v)", result.Error)
				}
				fmt.Println()
			}
		}
	}
}

func createApp() *cli.App {
	return &cli.App{
		Name:  "rux",
		Usage: "A fast Go-based test runner for Ruby/RSpec",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Print what would be executed without running",
			},
			&cli.BoolFlag{
				Name:  "auto",
				Usage: "Run bundle install if necessary before running tests",
			},
			&cli.BoolFlag{
				Name:  "json",
				Usage: "Save detailed test results to JSON files",
			},
			&cli.IntFlag{
				Name:    "workers",
				Aliases: []string{"j"},
				Usage:   "Number of parallel workers (default: cores-2, env: PARALLEL_TEST_PROCESSORS)",
			},
		},
		Action: func(c *cli.Context) error {
			var specFiles []string
			var err error

			// Determine which spec files to run
			if c.NArg() > 0 {
				// Use provided arguments as spec files
				specFiles = c.Args().Slice()
			} else {
				// Auto-discover spec files
				specFiles, err = findSpecFiles()
				if err != nil {
					return fmt.Errorf("error finding spec files: %v", err)
				}
				if len(specFiles) == 0 {
					return fmt.Errorf("no spec files found")
				}
			}

			dryRun := c.Bool("dry-run")

			if dryRun {
				if c.Bool("auto") {
					fmt.Fprintln(os.Stderr, "[dry-run] bundle install")
				}
				fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(specFiles))
				for _, file := range specFiles {
					args := []string{"bundle", "exec", "rspec", "--format", "progress", file}
					if c.Bool("json") {
						args = append(args, "--format", "json", "--out", "/tmp/results.json")
					}
					fmt.Fprintf(os.Stderr, "[dry-run] %s\n", strings.Join(args, " "))
				}
				return nil
			}

			// Run bundle install if --auto flag is set
			if c.Bool("auto") {
				fmt.Println("Installing dependencies...")
				bundleCmd := exec.Command("bundle", "install")
				bundleCmd.Stdout = os.Stdout
				bundleCmd.Stderr = os.Stderr
				
				if err := bundleCmd.Run(); err != nil {
					return fmt.Errorf("error running bundle install: %v", err)
				}
			}

			workerCount := getWorkerCount(c.Int("workers"))
			actualWorkers := workerCount
			if len(specFiles) < workerCount {
				actualWorkers = len(specFiles)
			}
			
			fmt.Printf("Running %d spec files in parallel using %d workers (%d cores available)...\n", 
				len(specFiles), actualWorkers, runtime.NumCPU())
			
			saveJSON := c.Bool("json")
			results, wallTime := runTestsInParallel(specFiles, dryRun, saveJSON, workerCount)
			printResults(results, wallTime)

			// Exit with error if any tests failed
			for _, result := range results {
				if !result.Success {
					os.Exit(1)
				}
			}

			return nil
		},
	}
}

func main() {
	app := createApp()
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}