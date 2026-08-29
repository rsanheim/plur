require "spec_helper"
require "open3"
require "timeout"

# watch/controller.go's attemptReload always calls stopRuns with forceAfter=0,
# so the shutdown wait it starts has no grace-period timer (unlike the
# non-terminal SIGINT path, which sets forceAfter=500ms). If the active job
# ignores SIGINT, stopRuns can block on doneChan forever with nothing left to
# wake it - SIGHUP-triggered reload is exactly the kind of unattended signal
# where nobody is present to send a second one and force it.
#
# These specs drive the real plur binary (PLUR_BINARY) rather than emulating
# the scheduler in isolation, and bound every wait with Timeout.timeout so a
# hang on main fails the example instead of hanging the suite.
RSpec.describe "plur watch reload hang" do
  include PlurWatchHelper

  # A job that traps SIGINT and swallows it, so plur's graceful interrupt()
  # can never make it exit on its own - only an explicit bound (or a second
  # external signal) can end the wait.
  def write_stubborn_job_project(project)
    project.join(".plur.toml").write(<<~TOML)
      use = "rspec"

      [job.rspec]
      cmd = ["ruby", "-e", #{stubborn_job_script.dump}]

      [[watch]]
      source = "spec/**/*_spec.rb"
      jobs = ["rspec"]
    TOML
  end

  def stubborn_job_script
    <<~RUBY.gsub("\n", "; ")
      trap('INT') { }
      File.write('runner.pid', Process.pid.to_s)
      loop { sleep 1 }
    RUBY
  end

  def process_pid(project)
    path = project.join("runner.pid")
    Integer(path.read, exception: false) if path.exist?
  end

  def process_alive?(pid)
    Process.kill(0, pid)
    true
  rescue Errno::ESRCH
    false
  end

  def kill_if_alive(pid, signal = "KILL")
    return unless pid && process_alive?(pid)

    Process.kill(signal, pid)
  rescue Errno::ESRCH
    nil
  end

  # Reads stdout/stderr non-blockingly until the given block returns true or
  # the bounded timeout elapses. Returns true if the condition was met,
  # false if the wait timed out - it never lets the caller hang.
  def pump_until(wait_thr, out, err, bound_seconds:)
    Timeout.timeout(bound_seconds) do
      loop do
        ready = IO.select([wait_thr[:stdout], wait_thr[:stderr]], nil, nil, 0.1)
        if ready
          ready[0].each do |io|
            data = io.read_nonblock(16_384)
            (io == wait_thr[:stdout]) ? out << data : err << data
          rescue IO::WaitReadable, EOFError
            nil
          end
        end

        return true if yield(out, err)
      end
    end
    false
  rescue Timeout::Error
    false
  end

  # Mirrors PlurWatchHelper#wait_for_watch_process: the Open3 wait thread
  # owns the waitpid call, so we join it rather than calling Process.wait
  # ourselves (which would race the wait thread for the same pid).
  def stop_plur(wait_thr)
    Process.kill("TERM", wait_thr.pid)
    Timeout.timeout(3) { wait_thr.value }
  rescue Errno::ESRCH
    nil
  rescue Timeout::Error
    begin
      Process.kill("KILL", wait_thr.pid)
    rescue Errno::ESRCH
      nil
    end
    wait_thr.value
  end

  it "bounds the SIGHUP reload wait when the active job ignores SIGINT" do
    with_temp_watch_project do |project|
      write_stubborn_job_project(project)

      Dir.chdir(project) do
        Open3.popen3(plur_binary, "--debug", "watch", "run", "--timeout", "30") do |stdin, stdout, stderr, wait_thr|
          out = +""
          err = +""
          io = {stdout: stdout, stderr: stderr}
          job_pid = nil

          begin
            entered = pump_until(io, out, err, bound_seconds: 10) { err.scan("s/self/live@").any? }
            raise "watch never reached ready state" unless entered

            stdin.puts("")
            job_ready = pump_until(io, out, err, bound_seconds: 10) { !!process_pid(project) }
            raise "watch job never started (runner.pid not written)" unless job_ready
            job_pid = process_pid(project)

            Process.kill("HUP", wait_thr.pid)

            reload_completed = pump_until(io, out, err, bound_seconds: 10) do
              err.scan("plur watch starting!").count >= 2
            end

            expect(reload_completed).to be(true), <<~MSG
              plur did not complete a SIGHUP reload within 10s while the active watch
              job ignored SIGINT (watch/controller.go:322 passes forceAfter=0 to
              stopRuns on the reload path, so the wait has no grace-period bound).

              stdout:
              #{out}

              stderr:
              #{err}
            MSG
          ensure
            stop_plur(wait_thr)
            kill_if_alive(job_pid)
            kill_if_alive(process_pid(project))
          end
        end
      end
    end
  end

  it "bounds the reload wait when 'reload' is typed while the active job ignores SIGINT" do
    with_temp_watch_project do |project|
      write_stubborn_job_project(project)

      Dir.chdir(project) do
        Open3.popen3(plur_binary, "--debug", "watch", "run", "--timeout", "30") do |stdin, stdout, stderr, wait_thr|
          out = +""
          err = +""
          io = {stdout: stdout, stderr: stderr}
          job_pid = nil

          begin
            entered = pump_until(io, out, err, bound_seconds: 10) { err.scan("s/self/live@").any? }
            raise "watch never reached ready state" unless entered

            stdin.puts("")
            job_ready = pump_until(io, out, err, bound_seconds: 10) { !!process_pid(project) }
            raise "watch job never started (runner.pid not written)" unless job_ready
            job_pid = process_pid(project)

            stdin.puts("reload")

            reload_completed = pump_until(io, out, err, bound_seconds: 10) do
              err.scan("plur watch starting!").count >= 2
            end

            expect(reload_completed).to be(true), <<~MSG
              plur did not complete a typed 'reload' within 10s while the active watch
              job ignored SIGINT (watch/controller.go:322 passes forceAfter=0 to
              stopRuns on the reload path, so the wait has no grace-period bound).

              stdout:
              #{out}

              stderr:
              #{err}
            MSG
          ensure
            stop_plur(wait_thr)
            kill_if_alive(job_pid)
            kill_if_alive(process_pid(project))
          end
        end
      end
    end
  end
end
