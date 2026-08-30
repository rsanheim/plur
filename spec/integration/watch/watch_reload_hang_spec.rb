require "spec_helper"

# Reload (SIGHUP or the typed command) must not wait forever on an active job
# that ignores SIGINT: attemptReload passes forceAfter=0 to stopRuns, so the
# reload wait has no grace-period bound. A completed reload re-execs plur,
# which logs a second "plur watch starting!"; a wedged plur never does, and
# the helper deadline kills it so a hang fails these examples instead of
# hanging the suite.
RSpec.describe "plur watch reload hang" do
  include PlurWatchHelper

  it "bounds the SIGHUP reload wait when the active job ignores SIGINT" do
    with_running_job_project(interrupt: "") do |project|
      expect_bounded_reload(project) do |process|
        Process.kill("HUP", process.pid)
      end
    end
  end

  it "bounds the reload wait when 'reload' is typed while the active job ignores SIGINT" do
    with_running_job_project(interrupt: "") do |project|
      expect_bounded_reload(project) do |process|
        process.stdin.puts("reload")
      end
    end
  end

  def expect_bounded_reload(project, &trigger_reload)
    entered = false
    reload_triggered = false

    result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
      if !entered && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
        process.stdin.puts("")
        entered = true
      elsif entered && !reload_triggered && process_pid(project)
        trigger_reload.call(process)
        reload_triggered = true
      elsif reload_triggered && reloaded?(process.err)
        :terminate
      end
    end

    expect(reloaded?(result.err)).to be(true), <<~MSG
      plur did not complete the reload within the deadline while the active
      job ignored SIGINT.

      stdout:
      #{result.out}

      stderr:
      #{result.err}
    MSG
  end

  def reloaded?(err)
    err.scan("plur watch starting!").count >= 2
  end
end
