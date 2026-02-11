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
    # Normalize watch directories to consistent order (lib before spec alphabetically)
    lines = str.lines
    watch_section_start = lines.index { |l| l.include?("Watch Directories:") }
    if watch_section_start
      # Find watch directory lines (lines with pattern "  dir/ (exists)")
      watch_lines = []
      i = watch_section_start + 1
      while i < lines.length && lines[i].match?(/^\s+\w+\/\s+\(exists\)/)
        watch_lines << lines[i]
        i += 1
      end
      # Sort watch directory lines alphabetically
      if watch_lines.any?
        sorted = watch_lines.sort
        sorted.each_with_index do |line, idx|
          lines[watch_section_start + 1 + idx] = line
        end
      end
    end
    str = lines.join

    str
      .gsub(/Plur Version:\s+.+/, "Plur Version:     [VERSION]")
      .gsub(/Build Date:\s+.+/, "Build Date:      [BUILD_DATE]")
      .gsub(/Git Commit:\s+.+/, "Git Commit:      [COMMIT]")
      .gsub(/Built By:\s+.+/, "Built By:        [BUILT_BY]")
      .gsub(/CLI Framework:\s+.+/, "CLI Framework:   [CLI_FRAMEWORK]")
      .gsub(/Operating System:\s+.+/, "Operating System: [OS]")
      .gsub(/Architecture:\s+.+/, "Architecture:     [ARCH]")
      .gsub(/CPU Count:\s+\d+/, "CPU Count:        [CPU_COUNT]")
      .gsub(/Go Version:\s+.+/, "Go Version:       [GO_VERSION]")
      .gsub(/Working Dir:\s+.+/, "Working Dir:      [WORKING_DIR]")
      .gsub(/Plur Binary:\s+.+/, "Plur Binary:       [PLUR_BINARY]")
      .gsub(/Binary Path:\s+.+/, "Binary Path:    [WATCHER_PATH]")
      .gsub(/^\s+Version:\s+.+/, "  Version:        [WATCHER_VERSION]")
      .gsub(/^\s+Platform:\s+.+/, "  Platform:       [WATCHER_PLATFORM]")
      .gsub(/Cache Directory:\s+.+/, "Cache Directory:  [CACHE_DIR]")
      .gsub(/Runtime Data:\s+.+/, "Runtime Data:     [RUNTIME_PATH]")
      .gsub(/Ruby Version:\s+.+/, "Ruby Version:   [RUBY_VERSION]")
      .gsub(/Bundler:\s+.+/, "Bundler:        [BUNDLER_VERSION]")
      .gsub(/RSpec:\s+.+/, "RSpec:          [RSPEC_VERSION]")
      .gsub(/- rspec-\w+\s+.+/, "- rspec-[COMPONENT] [VERSION]")
      .gsub(/HOME:\s+.+/, "HOME:                     [HOME_PATH]")
      .gsub(/Local Config:\s+.+/, "Local Config:   [LOCAL_CONFIG_PATH]")
      .gsub(/Global Config:\s+.+/, "Global Config:  [GLOBAL_CONFIG_PATH]")
      .gsub(/\s+- .*\.plur\.toml.*/, "    [CONFIG_FILE]")
      .gsub(/\s+Using defaults.*/, "    [DEFAULT_CONFIG_MESSAGE]")
      .gsub(/Source:\s+.+/, "Source: [CONFIG_SOURCE]")
      .gsub(/Command:\s+.+/, "Command: [COMMAND]")
      .gsub(/Active Job:\s+.+/, "Active Job: [JOB_NAME]")
      .gsub(/Target Pattern:\s+.+/, "Target Pattern: [TARGET_PATTERN]")
      .gsub(/Workers:\s+\d+/, "Workers: [WORKER_COUNT]")
      .gsub(/Color:\s+.+/, "Color: [COLOR_VALUE]")
      .gsub(/PARALLEL_TEST_PROCESSORS:\s+.+/, "PARALLEL_TEST_PROCESSORS: [PARALLEL_TEST_PROCESSORS]")
      .gsub(/FORCE_COLOR:\s+.+/, "FORCE_COLOR:              [FORCE_COLOR]")
      .gsub(/NO_COLOR:\s+.+/, "NO_COLOR:                 [NO_COLOR]")
      .gsub(/GOPATH:\s+.+/, "GOPATH:                   [GOPATH]")
      .gsub(/Debounce:\s+\d+ms/, "Debounce: [DEBOUNCE_MS]")
      .gsub("Watch Directories:", "Watch Directories:")
      .gsub(/\s+\w+\/\s+\(exists\)/, "    [DIR]/ (exists)")
  end

  def normalize_doctor_snapshot(snapshot)
    snapshot.merge("stdout" => normalize_doctor_output(snapshot.fetch("stdout", "")))
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
    Backspin.run(
      ["plur", "doctor"],
      name: "plur_doctor_golden",
      filter: ->(snapshot) { normalize_doctor_snapshot(snapshot) }
    )
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
      "Active Configuration Files:",
      "Active Settings:"
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
        use = "rspec"

        [job.rspec]
        cmd = ["bin/rspec", "--format", "progress", "{{target}}"]
        target_pattern = "spec/**/*_spec.rb"
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
      expect(stdout).to include("Active Configuration Files:")
      expect(stdout).to include(".plur.toml")

      # Check active configuration details
      expect(stdout).to include("Active Settings:")
      expect(stdout).to include("Workers:     8")
      expect(stdout).to include("Color:       false")
    end
  end
end
