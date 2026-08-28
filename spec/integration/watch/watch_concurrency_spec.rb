require "spec_helper"

# Watch runs disjoint work concurrently but does not run one target twice at
# the same time. These use log ordering to detect overlap, not timestamps.
RSpec.describe "plur watch run scheduling" do
  include PlurWatchHelper

  it "merges saves inside the debounce window into one run" do
    with_slow_job_project do |project|
      result = run_watch_until_finished(project, debounce: 500, expected_runs: 1) do
        touch(project.join("lib/calculator.rb"))
        touch(project.join("lib/validator.rb"))
      end

      expect(started_targets(result)).to eq([%w[spec/calculator_spec.rb spec/validator_spec.rb]])
    end
  end

  it "runs disjoint targets concurrently across debounce windows" do
    with_slow_job_project do |project|
      result = run_watch_until_finished(project, expected_runs: 2) do
        touch(project.join("lib/calculator.rb"))
        wait_for_child_job_start(project)
        touch(project.join("lib/validator.rb"))
      end

      expect(started_targets(result)).to contain_exactly(["spec/calculator_spec.rb"], ["spec/validator_spec.rb"])
      expect(overlapped?(result)).to be(true), timeline(result)
    end
  end

  it "skips an in-flight target when two sources map to it" do
    overlapping_watch = <<~TOML
      [[watch]]
      name = "user-overlap"
      source = "lib/user*.rb"
      targets = ["spec/models/user_spec.rb"]
      jobs = ["rspec"]
    TOML

    with_slow_job_project(extra_config: overlapping_watch) do |project|
      user = project.join("lib/user.rb")
      user_service = project.join("lib/user_service.rb")
      user.write("class User; end\n")
      user_service.write("class UserService; end\n")

      result = run_watch_until_finished(project, expected_runs: 1, expected_skips: 1) do
        touch(user)
        wait_for_child_job_start(project)
        touch(user_service)
      end

      expect(started_targets(result)).to eq([["spec/models/user_spec.rb"]])
      expect(skipped_targets(result)).to eq([["spec/models/user_spec.rb"]])
      expect(overlapped?(result)).to be(false), timeline(result)
      expect(result.out).to include("[plur] skipped spec/models/user_spec.rb reason=running")
    end
  end

  it "skips an in-flight target when one file is saved twice" do
    with_slow_job_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_watch_until_finished(project, expected_runs: 1, expected_skips: 1) do
        touch(spec_file)
        wait_for_child_job_start(project)
        touch(spec_file)
      end

      expect(started_targets(result)).to eq([["spec/calculator_spec.rb"]])
      expect(skipped_targets(result)).to eq([["spec/calculator_spec.rb"]])
      expect(overlapped?(result)).to be(false), timeline(result)
    end
  end

  it "runs only the free targets from a partially overlapping batch" do
    combined_watch = <<~TOML
      [[watch]]
      name = "combined"
      source = "lib/combined.rb"
      targets = ["spec/calculator_spec.rb", "spec/validator_spec.rb"]
      jobs = ["rspec"]
    TOML

    with_slow_job_project(extra_config: combined_watch) do |project|
      combined = project.join("lib/combined.rb")
      combined.write("# combined\n")

      result = run_watch_until_finished(project, expected_runs: 2, expected_skips: 1) do
        touch(project.join("spec/calculator_spec.rb"))
        wait_for_child_job_start(project)
        touch(combined)
      end

      expect(started_targets(result)).to contain_exactly(
        ["spec/calculator_spec.rb"],
        ["spec/validator_spec.rb"]
      )
      expect(skipped_targets(result)).to eq([["spec/calculator_spec.rb"]])
      expect(overlapped?(result)).to be(true), timeline(result)
    end
  end

  # Sleep makes overlap observable; the command ignores its target arguments.
  def with_slow_job_project(extra_config: "")
    with_temp_watch_project do |project|
      project.join(".plur.toml").write(<<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["ruby", "-e", "File.write('.plur-watch-job-started', 'started'); sleep #{job_sleep_seconds}"]

        #{extra_config}
      TOML

      yield project
    end
  end

  # Count completions so tests do not rely on --timeout after the last run.
  def run_watch_until_finished(project, expected_runs:, expected_skips: 0, debounce: nil, &block)
    run_plur_watch(
      dir: project,
      timeout: watch_timeout_seconds,
      debounce: debounce,
      stop_on: lambda { |process|
        process.err.scan("Finished job").count >= expected_runs &&
          process.err.scan("Skipped in-flight").count >= expected_skips
      },
      &block
    ).tap { |result| warn result.err if ENV["DEBUG"] }
  end

  # Log order is enough to detect overlap; timestamps are not.
  def run_events(result)
    pattern = /- INFO\s+- (Executing job|Finished job|Skipped in-flight) .*targets="\[([^\]]*)\]"/
    result.err.scan(pattern).map do |event, targets|
      {event: event, targets: targets.split(" ").sort}
    end
  end

  def started_targets(result)
    run_events(result).select { |e| e[:event] == "Executing job" }.map { |e| e[:targets] }
  end

  def skipped_targets(result)
    run_events(result).select { |e| e[:event] == "Skipped in-flight" }.map { |e| e[:targets] }
  end

  def overlapped?(result)
    live = 0
    run_events(result).select { |event| ["Executing job", "Finished job"].include?(event[:event]) }.each do |event|
      live += (event[:event] == "Executing job") ? 1 : -1
      return true if live > 1
    end
    false
  end

  def timeline(result)
    lines = run_events(result).map { |e| "  #{e[:event]} [#{e[:targets].join(" ")}]" }
    "run timeline was:\n#{lines.join("\n")}"
  end

  # Watch ignores mtime-only updates, so each recorded save changes content.
  def touch(path)
    path.write(path.read + "\n# touched by spec\n")
  end

  def wait_for_child_job_start(project)
    marker = project.join(".plur-watch-job-started")
    Timeout.timeout(watch_timeout_seconds) { sleep 0.01 until marker.exist? }
  end

  # Keep the job longer than the save gap, but short because these run in parallel.
  def job_sleep_seconds
    ENV["CI"] ? 3.0 : 0.8
  end

  def watch_timeout_seconds
    ENV["CI"] ? 30 : 15
  end
end
