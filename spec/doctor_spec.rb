require "spec_helper"

RSpec.describe "rux doctor command" do
  def run_rux_doctor(*args)
    cmd_array = %W[#{rux_binary} doctor]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  # Normalize dynamic values for consistent golden testing
  # Very general normalization - we only care about structure, not values
  def normalize_doctor_output(str)
    str
      # Version info - any version format
      .gsub(/Rux Version:\s+.+/, "Rux Version:     [VERSION]")
      .gsub(/Build Date:\s+.+/, "Build Date:      [BUILD_DATE]")
      .gsub(/Git Commit:\s+.+/, "Git Commit:      [COMMIT]")
      .gsub(/Built By:\s+.+/, "Built By:        [BUILT_BY]")
      
      # System info - normalize any numbers
      .gsub(/CPU Count:\s+\d+/, "CPU Count:        [CPU_COUNT]")
      .gsub(/Go Version:\s+.+/, "Go Version:       [GO_VERSION]")
      
      # Paths - normalize all paths
      .gsub(/Working Dir:\s+.+/, "Working Dir:      [WORKING_DIR]")
      .gsub(/Rux Binary:\s+.+/, "Rux Binary:       [RUX_BINARY]")
      .gsub(/Binary Path:\s+.+/, "Binary Path:    [WATCHER_PATH]")
      .gsub(/Cache Directory:\s+.+/, "Cache Directory:  [CACHE_DIR]")
      .gsub(/Runtime Data:\s+.+/, "Runtime Data:     [RUNTIME_PATH]")
      
      # Ruby environment - very general
      .gsub(/Ruby Version:\s+.+/, "Ruby Version:   [RUBY_VERSION]")
      .gsub(/Bundler:\s+.+/, "Bundler:        [BUNDLER_VERSION]")
      .gsub(/RSpec:\s+.+/, "RSpec:          [RSPEC_VERSION]")
      .gsub(/- rspec-\w+\s+.+/, "- rspec-[COMPONENT] [VERSION]")
      
      # Environment variables - normalize values but keep keys
      .gsub(/HOME:\s+.+/, "HOME:                     [HOME_PATH]")
  end

  it "displays diagnostic information" do
    stdout, stderr, status = run_rux_doctor

    expect(status.exitstatus).to eq(0)
    expect(stderr).to be_empty

    # Basic structure checks
    expect(stdout).to include("Rux Doctor")
    expect(stdout).to include("Rux Version:")
    expect(stdout).to include("Operating System:")
    expect(stdout).to include("Ruby Environment:")
    expect(stdout).to include("File Watcher:")
    expect(stdout).to include("Environment Variables:")
  end

  it "shows e-dant/watcher availability" do
    stdout, _stderr, _status = run_rux_doctor

    expect(stdout).to match(/Status:\s+Available/)
    expect(stdout).to match(/Binary Path:\s+/)
  end

  it "produces consistent output using Backspin golden testing", :skip_if_ci do
    stdout_matcher = ->(record, actual) {
      normalized_recorded = normalize_doctor_output(record)
      normalized_actual = normalize_doctor_output(actual)

      normalized_recorded == normalized_actual
    }

    Backspin.run!("rux_doctor_golden",
      matcher: {stdout: stdout_matcher}) do
      run_rux_doctor
    end
  end

  it "includes all expected sections in output" do
    stdout, _stderr, _status = run_rux_doctor

    expected_sections = [
      "Rux Doctor",
      "Rux Version:",
      "Build Date:",
      "Git Commit:",
      "Built By:",
      "Operating System:",
      "Architecture:",
      "CPU Count:",
      "Go Version:",
      "Working Dir:",
      "Rux Binary:",
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
      "GOPATH:"
    ]

    expected_sections.each do |section|
      expect(stdout).to include(section), "Expected to find '#{section}' in doctor output"
    end
  end
end
