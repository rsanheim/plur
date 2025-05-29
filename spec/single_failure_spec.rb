RSpec.describe "single failure" do
  def fixture_path(name)
    File.join(__dir__, "..", "test_fixtures", name)
  end

  def capture(cmd_array)
    Open3.capture3(*cmd_array)
  end

  def run_rux(file_or_glob, *args)
    cmd_array = %W(#{rux_binary} #{file_or_glob})
    cmd_array += args if args.any?
    capture(cmd_array)
  end

  def run_rspec(file_or_glob, *args)
    cmd_array = %W(bundle exec rspec #{file_or_glob})
    cmd_array += args if args.any?
    capture(cmd_array)
  end

  it "prints correct summary counts" do
    chdir fixture_path("failing_specs") do
      stdout, stderr, status = run_rux("--no-color", "spec/single_failure_spec.rb")
      expect(status.exitstatus).to eq(1)
      expect(stdout).to match(/1 failure\b/)
      expect(stderr).to match(/1 file across (\d+) workers/)
    end
  end

  it "shows filtered backtrace same as rspec" do
    chdir fixture_path("failing_specs") do
      rspec_out, rspec_err, rspec_status = run_rspec("spec/single_failure_spec.rb", "--tty", "--force-color")
      expect(rspec_status.exitstatus).to eq(1)

      backtrace_line = rspec_out.split("\n").find { |line| line.include?("./spec/single_failure_spec.rb:6") } # remove color codes

      stdout, stderr, status = run_rux("spec/single_failure_spec.rb")
      expect(status.exitstatus).to eq(1)
      expect(stdout).to include(backtrace_line)
    end
  end

  # Replace
  #   Finished in 0.00761 seconds (files took 0.04367 seconds to load)\n
  # with
  #   Finished in [fake-time] seconds (files took [fake-time] seconds to load)
  def make_summary_line_consistent(str)
    str.gsub(/Finished in \d+\.\d+ seconds \(files took \d+\.\d+ seconds to load\)/, 
             "Finished in [fake-time] seconds (files took [fake-time] seconds to load)")
  end

  it "matches rspec colorized output" do
    chdir fixture_path("failing_specs") do
      rspec_out, rspec_err, rspec_status = run_rspec("spec/single_failure_spec.rb", "--force-color", "--tty")
      expect(rspec_status.exitstatus).to eq(1)

      stdout, stderr, status = run_rux("spec/single_failure_spec.rb")
      expect(status.exitstatus).to eq(1)

      stdout_lines = make_summary_line_consistent(stdout).lines
      stdout_without_preamble = stdout_lines[2..-1]
      rspec_stdout_lines = make_summary_line_consistent(rspec_out).lines

      expect(stdout_without_preamble).to eq(rspec_stdout_lines)
    end
  end
end