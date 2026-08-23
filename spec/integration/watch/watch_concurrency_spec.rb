require "spec_helper"

# Specification for watch's in-flight guard: saves in one debounce window
# batch merge into one run; a save of an already-running target is skipped
# and reported, while disjoint targets across debounce windows still run
# concurrently. They use log ordering to detect overlap, not timestamps.
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
        sleep save_gap_seconds
        touch(project.join("lib/validator.rb"))
      end

      expect(started_targets(result)).to contain_exactly(["spec/calculator_spec.rb"], ["spec/validator_spec.rb"])
      expect(overlapped?(result)).to be(true), timeline(result)
    end
  end

  # These two events select one target; the second is skipped while the first runs.
  it "skips in-flight target when two sources map to it" do
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

      result = run_watch_until_finished(project, expected_runs: 1) do
        touch(user)
        sleep save_gap_seconds
        touch(user_service)
      end

      expect(started_targets(result)).to eq([["spec/models/user_spec.rb"]])
      expect(result.out).to include("[plur] skipped spec/models/user_spec.rb reason=running")
    end
  end

  # A second save of an in-flight file is skipped, not run concurrently.
  it "skips in-flight target when one file is saved twice" do
    with_slow_job_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_watch_until_finished(project, expected_runs: 1) do
        touch(spec_file)
        sleep save_gap_seconds
        touch(spec_file)
      end

      expect(started_targets(result)).to eq([["spec/calculator_spec.rb"]])
      expect(result.out).to include("[plur] skipped spec/calculator_spec.rb reason=running")
    end
  end

  # After a run finishes, a fresh save should start a new run.
  it "runs the same target again after in-flight run finishes" do
    with_slow_job_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_watch_until_finished(project, expected_runs: 2) do
        touch(spec_file)
        sleep save_gap_seconds
        touch(spec_file)
        # Wait for the first run to finish and release its claim. Doubling
        # job_sleep_seconds (rather than a small fixed addition) gives real
        # headroom under CPU contention -- e.g. CI running this suite's own
        # workers concurrently -- while still leaving the overall example
        # comfortably inside watch_timeout_seconds.
        sleep(job_sleep_seconds * 2)
        touch(spec_file)
      end

      expect(started_targets(result)).to eq([["spec/calculator_spec.rb"], ["spec/calculator_spec.rb"]])
      # Verify the skip happened between first and second successful run
      events = run_events(result)
      expect(events[0][:event]).to eq("Executing job")
      expect(events[1][:event]).to eq("Skipped in-flight")
      expect(events[2][:event]).to eq("Finished job")
      expect(events[3][:event]).to eq("Executing job")
    end
  end

  # A batch that re-saves an in-flight target alongside a new one narrows to
  # just the new target; the in-flight one is dropped and reported.
  it "partially narrows a batch that overlaps an in-flight run" do
    with_slow_job_project do |project|
      calc = project.join("lib/calculator.rb")
      validator = project.join("lib/validator.rb")

      result = run_watch_until_finished(project, expected_runs: 2) do
        # First batch: calculator only.
        touch(calc)
        sleep save_gap_seconds
        # Second batch: calculator (already in flight) plus validator (new).
        touch(calc)
        touch(validator)
      end

      started = started_targets(result)
      expect(started[0]).to eq(["spec/calculator_spec.rb"])
      expect(started[1]).to eq(["spec/validator_spec.rb"])
      expect(result.out).to include("[plur] skipped spec/calculator_spec.rb reason=running")
    end
  end

  # The no-targets lane is independent of the targeted lane: a no-targets run
  # only suppresses a duplicate no-targets run of the same job.
  it "skips a duplicate no-targets run of the same job" do
    no_targets_watch = <<~TOML
      [[watch]]
      name = "readme-triggers-full"
      source = "README.md"
      no_targets = true
      jobs = ["rspec"]
    TOML

    with_slow_job_project(extra_config: no_targets_watch) do |project|
      readme = project.join("README.md")

      result = run_watch_until_finished(project, expected_runs: 1) do
        touch(readme)
        sleep save_gap_seconds
        touch(readme)
      end

      expect(started_targets(result)).to eq([[]])
      expect(result.out).to include("[plur] skipped rspec reason=running")
    end
  end

  # A no-targets run in flight never blocks a targeted run of the same job,
  # and a targeted run in flight never blocks a no-targets run.
  it "runs a no-targets run and a targeted run of the same job concurrently" do
    no_targets_watch = <<~TOML
      [[watch]]
      name = "readme-triggers-full"
      source = "README.md"
      no_targets = true
      jobs = ["rspec"]
    TOML

    with_slow_job_project(extra_config: no_targets_watch) do |project|
      readme = project.join("README.md")
      calc = project.join("lib/calculator.rb")

      result = run_watch_until_finished(project, expected_runs: 2) do
        touch(readme)
        sleep save_gap_seconds
        touch(calc)
      end

      started = started_targets(result)
      expect(started).to contain_exactly([], ["spec/calculator_spec.rb"])
      expect(overlapped?(result)).to be(true), timeline(result)
      expect(result.out).not_to include("skipped")
    end
  end

  # Sleep makes overlap observable; the command ignores its target arguments.
  def with_slow_job_project(extra_config: "")
    with_temp_watch_project do |project|
      project.join(".plur.toml").write(<<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["ruby", "-e", "sleep #{job_sleep_seconds}"]

        #{extra_config}
      TOML

      yield project
    end
  end

  # Count completions so tests do not rely on --timeout after the last run.
  def run_watch_until_finished(project, expected_runs:, debounce: nil, &block)
    run_plur_watch(
      dir: project,
      timeout: watch_timeout_seconds,
      debounce: debounce,
      stop_on: ->(process) { process.err.scan("Finished job").count >= expected_runs },
      &block
    )
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

  def overlapped?(result)
    live = 0
    run_events(result).each do |event|
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

  # Keep the job longer than the save gap, but short because these run in parallel.
  def job_sleep_seconds
    ENV["CI"] ? 3.0 : 0.8
  end

  def save_gap_seconds
    0.15
  end

  def watch_timeout_seconds
    ENV["CI"] ? 30 : 15
  end
end
