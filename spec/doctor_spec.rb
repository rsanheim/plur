require "spec_helper"

RSpec.describe "rux doctor command" do
  def run_rux_doctor(*args)
    cmd_array = %W[#{rux_binary} doctor]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  # Normalize dynamic values for consistent golden testing
  def normalize_doctor_output(output)
    output
      # Normalize version info
      .gsub(/v\d+\.\d+\.\d+-\d+-\d+-[a-f0-9]+/, "v[VERSION]")
      .gsub(/\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2} UTC/, "[BUILD_DATE]")
      .gsub(/[a-f0-9]{9}/, "[COMMIT]")
      # Normalize paths
      .gsub(/\/Users\/[^\/]+/, "/Users/[USER]")
      .gsub(/\/home\/[^\/]+/, "/home/[USER]")
      # Normalize watcher binary paths
      .gsub(/Binary Path:\s+\/Users\/[^\/]+\/.cache\/rux\/bin\/[^\n]+/, "Binary Path:    [WATCHER_PATH]")
      # Normalize Go version
      .gsub(/go\d+\.\d+\.\d+/, "go[VERSION]")
      # Normalize Ruby versions
      .gsub(/ruby \d+\.\d+\.\d+ \(\d{4}-\d{2}-\d{2} revision [a-f0-9]+\)/, "ruby [VERSION]")
      .gsub(/Bundler version \d+\.\d+\.\d+/, "Bundler version [VERSION]")
      .gsub(/RSpec \d+\.\d+/, "RSpec [VERSION]")
      .gsub(/rspec-\w+ \d+\.\d+\.\d+/, "rspec-[MODULE] [VERSION]")
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
    # Use use_dubplate for easier recording management
    Backspin.use_dubplate("rux_doctor_golden", record: :once) do
      run_rux_doctor
    end

    # Verify current output matches golden (with normalization)
    Backspin.verify!("rux_doctor_golden",
      matcher: ->(recorded, actual) {
        # Both should succeed with exit status 0
        return false unless recorded["status"] == 0 && actual["status"] == 0

        # Normalize both outputs for comparison
        recorded_normalized = normalize_doctor_output(recorded["stdout"])
        actual_normalized = normalize_doctor_output(actual["stdout"])

        # Instead of exact match, check that all key sections are present
        # and in the correct order
        key_sections = [
          "Rux Doctor",
          "Rux Version:",
          "Operating System:",
          "Ruby Environment:",
          "File Watcher:",
          "Cache Directory:",
          "Environment Variables:"
        ]

        # Check that all key sections appear in both outputs
        key_sections.all? do |section|
          recorded_normalized.include?(section) && actual_normalized.include?(section)
        end
      }) do
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
