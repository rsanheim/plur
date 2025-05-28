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

  fit "shows filtered backtrace same as rspec" do
    chdir fixture_path("failing_specs") do
      rspec_out, rspec_err, rspec_status = run_rspec("spec/single_failure_spec.rb")
      expect(rspec_status.exitstatus).to eq(1)

      backtrace_line = rspec_out.split("\n").find { |line| line.include?("# ./spec/single_failure_spec.rb") }

      stdout, stderr, status = run_rux("spec/single_failure_spec.rb")
      expect(status.exitstatus).to eq(1)
      expect(stdout).to include(backtrace_line)
    end
  end

  it "matches rspec colorized output" do
    chdir fixture_path("failing_specs") do
      rspec_out, rspec_err, rspec_status = run_rspec("spec/single_failure_spec.rb", "--force-color", "--tty")
      expect(rspec_status.exitstatus).to eq(1)

      stdout, stderr, status = run_rux("spec/single_failure_spec.rb")
      expect(status.exitstatus).to eq(1)

      stdout_lines = stdout.split("\n")
      pp stdout_lines[0..2]
      stdout_without_preamble = stdout_lines[2..-1].join("\n")

      expect(stdout_without_preamble).to eq(rspec_out)
    end
  end
end