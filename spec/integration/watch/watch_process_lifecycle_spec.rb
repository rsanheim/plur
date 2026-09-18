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
end
