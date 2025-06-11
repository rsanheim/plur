require "spec_helper"

RSpec.describe "rux doctor command" do
  def run_rux_doctor(*args)
    cmd_array = %W[#{rux_binary} doctor]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  # Normalize dynamic values for consistent golden testing
  # Using loose regex patterns here deliberately, its good enough
  def normalize_doctor_output(str)
    str
      .gsub(/v[\d\.\-\w\+]+/, "v[VERSION]")
      .gsub(/(Build Date:\s+).+/, '\1[BUILD_DATE]')
      .gsub(/(Git Commit:\s+).+/, '\1[COMMIT]')
      .gsub(/(Built By:\s+).+/, '\1[BUILT_BY]')
      # Normalize paths
      .gsub(/\/Users\/[^\/]+/, "/Users/[USER]")
      .gsub(/\/home\/[^\/]+/, "/home/[USER]")
      # Normalize watcher binary paths
      .gsub(/Binary Path:\s+\/Users\/[^\/]+\/.cache\/rux\/bin\/[^\n]+/, "Binary Path:    [WATCHER_PATH]")
      # Normalize Go version
      .gsub(/go[\d\.]+/, "go[VERSION]")
      # Normalize Ruby versions - looser patterns
      .gsub(/ruby [\d\.\w\s\(\)\-]+/, "ruby [VERSION]")
      .gsub(/Bundler version [\d\.]+/, "Bundler version [VERSION]")
      .gsub(/RSpec [\d\.]+/, "RSpec [VERSION]")
      .gsub(/- rspec-\w+ [\d\.]+/, "- rspec-[COMPONENT] [VERSION]")
      # Normalize runtime data hash
      .gsub(/[a-f0-9]{8}\.json/, "[HASH].json")
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

  it "shows watcher availability on supported platforms" do
    stdout, _stderr, _status = run_rux_doctor

    if RUBY_PLATFORM.include?("darwin") && RUBY_PLATFORM.include?("arm64")
      expect(stdout).to include("Status:         Available")
      expect(stdout).to include("Binary Path:")
    else
      expect(stdout).to match(/Status:\s+Not available/)
    end
  end

  it "produces consistent output using Backspin golden testing" do
    stdout_matcher = ->(record, actual) {
      normalized_recorded = normalize_doctor_output(record)
      normalized_actual = normalize_doctor_output(actual)

      normalized_recorded == normalized_actual
    }

    result = Backspin.run!("rux_doctor_golden", 
      matcher: { stdout: stdout_matcher}) do
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
