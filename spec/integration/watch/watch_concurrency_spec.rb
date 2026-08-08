require "spec_helper"

# Characterization specs for how plur watch schedules runs today.
#
# The debouncer batches by source path inside a single window, then fires each
# batch on its own goroutine with nothing serializing them. That produces two
# distinct behaviors depending on whether saves land inside the debounce
# window, and the cross-window case runs jobs concurrently regardless of
# whether their targets overlap.
#
# These specs pin all of it -- the concurrency we want to keep and the
# overlapping runs we want to stop -- so a scheduling change has a baseline to
# diff against. Specs describing behavior slated to change say so inline.
#
# Overlap is read from the "Executing job" / "Finished job" pair plur logs
# around each run. Two starts before either finish is what overlap looks like;
# no timestamp math, just ordering.
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

  # Behavior slated to change: these two runs share every target, so the second
  # should wait for the first rather than run the same spec file twice at once.
  it "runs the same target concurrently when two sources map to it" do
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

      result = run_watch_until_finished(project, expected_runs: 2) do
        touch(user)
        sleep save_gap_seconds
        touch(user_service)
      end

      expect(started_targets(result)).to eq([["spec/models/user_spec.rb"], ["spec/models/user_spec.rb"]])
      expect(overlapped?(result)).to be(true), timeline(result)
    end
  end

  # Behavior slated to change: saving one file twice while its run is still in
  # flight starts a second identical run alongside the first.
  it "runs the same target concurrently when one file is saved twice" do
    with_slow_job_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_watch_until_finished(project, expected_runs: 2) do
        touch(spec_file)
        sleep save_gap_seconds
        touch(spec_file)
      end

      expect(started_targets(result)).to eq([["spec/calculator_spec.rb"], ["spec/calculator_spec.rb"]])
      expect(overlapped?(result)).to be(true), timeline(result)
    end
  end

  # Copies the default-ruby fixture and swaps its rspec job for one that just
  # sleeps, so runs last long enough to observe whether they overlap. Targets
  # are appended as argv and ignored.
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

  # Runs watch until plur has logged expected_runs completions, so a spec never
  # depends on --timeout landing after the last run.
  def run_watch_until_finished(project, expected_runs:, debounce: nil, &block)
    run_plur_watch(
      dir: project,
      timeout: watch_timeout_seconds,
      debounce: debounce,
      until_condition: ->(process) { process.err.scan("Finished job").count >= expected_runs },
      &block
    )
  end

  # The start and finish lines each run logs, in the order plur emitted them.
  def run_events(result)
    pattern = /- INFO\s+- (Executing job|Finished job) .*targets="\[([^\]]*)\]"/
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

  # Watch only reacts to create and modify, not bare mtime bumps, so a
  # recorded save has to change content.
  def touch(path)
    path.write(path.read + "\n# touched by spec\n")
  end

  # A run has to outlast the gap between saves for overlap to be observable at
  # all; the margin between the two is the slack these specs run on. Kept as
  # short as that slack allows, since the suite runs specs in parallel and
  # these hold watch processes open for their whole duration.
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
