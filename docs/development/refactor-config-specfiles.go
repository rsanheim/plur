package main

// Refactored approach: Config holds only global flags, SpecFiles moved to command context

// Config now only holds truly global configuration
type Config struct {
	Auto         bool
	ColorOutput  bool
	ConfigPaths  *ConfigPaths
	DryRun       bool
	TraceEnabled bool
	WorkerCount  int
	// REMOVED: SpecFiles - not global!
}

// TestExecutor now takes specFiles as a parameter
type TestExecutor struct {
	config         *Config
	specFiles      []string  // Passed in, not from config
	runtimeTracker *RuntimeTracker
}

// BuildConfig now only builds from global flags
func BuildConfig(ctx *cli.Context, paths *ConfigPaths) (*Config, error) {
	return &Config{
		Auto:         ctx.Bool("auto"),
		ColorOutput:  shouldUseColor(ctx),
		ConfigPaths:  paths,
		DryRun:       ctx.Bool("dry-run"),
		TraceEnabled: ctx.Bool("trace"),
		WorkerCount:  GetWorkerCount(ctx.Int("n")),
	}, nil
}

// Main app structure with separated concerns
func createApp() *cli.App {
	return &cli.App{
		Name:    "rux",
		Usage:   "A fast Go-based test runner for Ruby/RSpec",
		Version: GetVersionInfo(),
		Before: func(ctx *cli.Context) error {
			// Initialize logging
			debug := ctx.Bool("debug") || os.Getenv("RUX_DEBUG") == "1"
			InitLogger(ctx.Bool("verbose"), debug)
			
			// Build global config from global flags only
			var err error
			ruxConfig, err = BuildConfig(ctx, configPaths)
			if err != nil {
				return fmt.Errorf("failed to initialize config: %v", err)
			}
			Logger.Debug("global config initialized", "config", ruxConfig)
			
			return nil
		},
		Action: func(ctx *cli.Context) error {
			// Main command discovers spec files HERE, not in Before
			specFiles, err := discoverSpecFiles(ctx)
			if err != nil {
				return err
			}
			
			// Create executor with config AND spec files
			executor := &TestExecutor{
				config:         ruxConfig,
				specFiles:      specFiles,
				runtimeTracker: NewRuntimeTracker(),
			}
			
			return executor.Execute()
		},
		Commands: []*cli.Command{
			{
				Name:  "watch",
				Usage: "Watch for file changes and run tests automatically",
				Action: func(ctx *cli.Context) error {
					// Watch doesn't use spec files from args at all
					// It discovers files through file system watching
					return WatchSpecFiles(ruxConfig, ctx.Int("timeout"), ctx.Int("debounce"))
				},
			},
			{
				Name:  "doctor",
				Usage: "Diagnose common issues and verify installation",
				Action: func(ctx *cli.Context) error {
					// Doctor doesn't need spec files
					return RunDoctor(ruxConfig.ColorOutput)
				},
			},
		},
	}
}

// Updated TestExecutor methods
func (e *TestExecutor) Execute() error {
	fmt.Printf("rux version %s\n", GetVersionInfo())
	
	if e.config.DryRun {
		return e.executeDryRun()
	}
	
	return e.executeTests()
}

func (e *TestExecutor) executeDryRun() error {
	if e.config.Auto {
		fmt.Fprintln(os.Stderr, "[dry-run] bundle install")
	}
	
	// Use e.specFiles instead of e.config.SpecFiles
	fmt.Fprintf(os.Stderr, "[dry-run] Found %d spec files, running in parallel:\n", len(e.specFiles))
	
	// ... rest of method using e.specFiles
}

// Alternative: If you want to avoid modifying TestExecutor, 
// create a TestContext that combines Config + spec files
type TestContext struct {
	Config    *Config
	SpecFiles []string
}

func NewTestExecutorWithContext(ctx *TestContext) *TestExecutor {
	return &TestExecutor{
		config:         ctx.Config,
		specFiles:      ctx.SpecFiles,
		runtimeTracker: NewRuntimeTracker(),
	}
}

// Or even simpler: Just pass specFiles to Execute
func (e *TestExecutor) ExecuteWithSpecs(specFiles []string) error {
	e.specFiles = specFiles
	return e.Execute()
}