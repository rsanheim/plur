require "spec_helper"

RSpec.describe "plur watch process lifecycle" do
  include PlurWatchHelper

  it "stops a manual run on exit" do
    with_running_job_project do |project|
      entered = false
      exited = false

      result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
        if !entered && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
          process.stdin.puts("")
          entered = true
        elsif entered && !exited && process_pid(project)
          process.stdin.puts("exit")
          process.close_stdin
          exited = true
        end
      end

      expect(result).to be_success
      expect(result.out).to include("Exiting watch mode...")
      expect_process_gone(project)
    end
  end

  it "stops a file-triggered run on SIGTERM" do
    with_running_job_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_plur_watch(
        dir: project,
        timeout: 10,
        stop_on: ->(_process) { process_pid(project) }
      ) do
        spec_file.write(spec_file.read + "\n# lifecycle spec\n")
      end

      expect(result).to be_success
      expect(result.out).to include("Received SIGTERM")
      expect_process_gone(project)
    end
  end

  it "forwards a non-terminal SIGINT and exits without another signal" do
    with_running_job_project(interrupt: "File.write('runner.interrupted', 'yes')") do |project|
      entered = false
      interrupted = false

      result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
        if !entered && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
          process.stdin.puts("")
          entered = true
        elsif entered && !interrupted && process_pid(project)
          Process.kill("INT", process.pid)
          interrupted = true
        end
      end

      expect(result).to be_success
      expect(result.out).to include("Received SIGINT, stopping active jobs...")
      expect(result.out).to include("Shutdown grace period elapsed, forcing active jobs to stop...")
      expect(project.join("runner.interrupted")).to exist
      expect_process_gone(project)
    end
  end

  def with_running_job_project(interrupt: "exit 130")
    with_temp_watch_project do |project|
      script = <<~RUBY.gsub("\n", "; ")
        trap('INT') { #{interrupt} }
        File.write('runner.pid', Process.pid.to_s)
        loop { sleep 1 }
      RUBY

      project.join(".plur.toml").write(<<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["ruby", "-e", #{script.dump}]

        [[watch]]
        source = "spec/**/*_spec.rb"
        jobs = ["rspec"]
      TOML

      yield project
    ensure
      pid = process_pid(project)
      if pid && process_alive?(pid)
        begin
          Process.kill("KILL", pid)
        rescue Errno::ESRCH
          nil
        end
      end
    end
  end

  def process_pid(project)
    path = project.join("runner.pid")
    Integer(path.read, exception: false) if path.exist?
  end

  def expect_process_gone(project)
    pid = process_pid(project)
    expect(pid).not_to be_nil
    Timeout.timeout(3) { sleep 0.02 while process_alive?(pid) }
  end

  def process_alive?(pid)
    Process.kill(0, pid)
    true
  rescue Errno::ESRCH
    false
  end
end
