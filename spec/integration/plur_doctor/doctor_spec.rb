require "spec_helper"

RSpec.describe "plur doctor command" do
  def run_plur_doctor(*args)
    # Use Open3 directly to match Backspin's expected format
    cmd_array = ["plur", "doctor"]
    cmd_array += args if args.any?

    Open3.capture3(*cmd_array)
  end

  # Normalize dynamic values for consistent golden testing
  # Very general normalization - we only care about structure, not values
  def normalize_doctor_output(str)
    str
      .gsub(/Plur Version:\s+.+/, "Plur Version:     [VERSION]")
      .gsub(/Build Date:\s+.+/, "Build Date:      [BUILD_DATE]")
      .gsub(/Git Commit:\s+.+/, "Git Commit:      [COMMIT]")
      .gsub(/Built By:\s+.+/, "Built By:        [BUILT_BY]")
      .gsub(/CLI Framework:\s+.+/, "CLI Framework:   [CLI_FRAMEWORK]")
      .gsub(/CPU Count:\s+\d+/, "CPU Count:        [CPU_COUNT]")
      .gsub(/Go Version:\s+.+/, "Go Version:       [GO_VERSION]")
      .gsub(/Working Dir:\s+.+/, "Working Dir:      [WORKING_DIR]")
      .gsub(/Plur Binary:\s+.+/, "Plur Binary:       [PLUR_BINARY]")
      .gsub(/Binary Path:\s+.+/, "Binary Path:    [WATCHER_PATH]")
      .gsub(/Cache Directory:\s+.+/, "Cache Directory:  [CACHE_DIR]")
      .gsub(/Runtime Data:\s+.+/, "Runtime Data:     [RUNTIME_PATH]")
      .gsub(/Ruby Version:\s+.+/, "Ruby Version:   [RUBY_VERSION]")
      .gsub(/Bundler:\s+.+/, "Bundler:        [BUNDLER_VERSION]")
      .gsub(/RSpec:\s+.+/, "RSpec:          [RSPEC_VERSION]")
      .gsub(/- rspec-\w+\s+.+/, "- rspec-[COMPONENT] [VERSION]")
      .gsub(/HOME:\s+.+/, "HOME:                     [HOME_PATH]")
      .gsub(/Local Config:\s+.+/, "Local Config:   [LOCAL_CONFIG_PATH]")
      .gsub(/Global Config:\s+.+/, "Global Config:  [GLOBAL_CONFIG_PATH]")
      .gsub(/Source:\s+.+/, "Source: [CONFIG_SOURCE]")
      .gsub(/Command:\s+.+/, "Command: [COMMAND]")
      .gsub(/Workers:\s+\d+/, "Workers: [WORKER_COUNT]")
      .gsub(/Color:\s+.+/, "Color: [COLOR_VALUE]")
      .gsub(/Debounce:\s+\d+ms/, "Debounce: [DEBOUNCE_MS]")
      .gsub("Watch Directories:", "Watch Directories:")
      .gsub(/\s+\w+\/\s+\(exists\)/, "    [DIR]/ (exists)")
  end

  it "displays diagnostic information" do
    stdout, stderr, status = run_plur_doctor

    expect(status.exitstatus).to eq(0)
    expect(stderr).to be_empty

    # Basic structure checks
    expect(stdout).to include("Plur Doctor")
    expect(stdout).to include("Plur Version:")
    expect(stdout).to include("Operating System:")
    expect(stdout).to include("Ruby Environment:")
    expect(stdout).to include("File Watcher:")
    expect(stdout).to include("Environment Variables:")
  end

  it "shows e-dant/watcher availability" do
    stdout, _stderr, _status = run_plur_doctor

    expect(stdout).to match(/Status:\s+Available/)
    expect(stdout).to match(/Binary Path:\s+/)
  end

  it "produces consistent output using Backspin golden testing", :skip_if_ci do
    stdout_matcher = ->(record, actual) {
      normalized_recorded = normalize_doctor_output(record)
      normalized_actual = normalize_doctor_output(actual)

      normalized_recorded == normalized_actual
    }

    Backspin.run!("plur_doctor_golden",
      matcher: {stdout: stdout_matcher}) do
      run_plur_doctor
    end
  end

  it "includes all expected sections in output" do
    stdout, _stderr, _status = run_plur_doctor

    expected_sections = [
      "Plur Doctor",
      "Plur Version:",
      "Build Date:",
      "Git Commit:",
      "Built By:",
      "Operating System:",
      "Architecture:",
      "CPU Count:",
      "Go Version:",
      "Working Dir:",
      "Plur Binary:",
      "Ruby Environment:",
      "Ruby Version:",
      "Bundler:",
      "RSpec:",
      "File Watcher:",
      "Cache Directory:",
      "Runtime Data:",
      "Environment Variables:",
      "PARALLEL_TEST_PROCESSORS:",
      "FORCE_COLOR:",
      "NO_COLOR:",
      "HOME:",
      "GOPATH:",
      "Configuration:",
      "Local Config:",
      "Global Config:",
      "Active Configuration:"
    ]

    expected_sections.each do |section|
      expect(stdout).to include(section), "Expected to find '#{section}' in doctor output"
    end
  end

  context "with configuration file" do
    let(:test_dir) { Dir.mktmpdir }

    before do
      File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
        workers = 8
        color = false
        command = "rspec --no-color"
        
        [spec]
        command = "bin/rspec --format progress"
        
        [watch.run]
        command = "bundle exec rspec"
        debounce = 500
      TOML
    end

    after do
      FileUtils.remove_entry(test_dir)
    end

    it "displays configuration information" do
      stdout, _stderr, _status = Dir.chdir(test_dir) do
        run_plur_doctor
      end

      # Check configuration section appears
      expect(stdout).to include("Configuration:")
      expect(stdout).to include("Local Config:")
      expect(stdout).to include(".plur.toml (valid")

      # Check active configuration details
      expect(stdout).to include("Active Configuration:")
      expect(stdout).to include("Source: local .plur.toml")
      expect(stdout).to include("Workers: 8")
      expect(stdout).to include("Color: false")
      expect(stdout).to include("Command: rspec --no-color")

      # Check command-specific sections
      expect(stdout).to include("[spec] section:")
      expect(stdout).to include("Command: bin/rspec --format progress")
      expect(stdout).to include("[watch.run] section:")
      expect(stdout).to include("Debounce: 500ms")
    end
  end
end
