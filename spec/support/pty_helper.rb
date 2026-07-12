require "pty"

# Runs a command under a real pseudo-terminal so specs can assert TTY-side
# behavior (plur's auto color mode). Pipe-side behavior needs no PTY — the
# regular run_plur helper already runs through pipes.
module PtyHelper
  PTY_AVAILABLE = begin
    PTY.spawn("true") { |_out, _in, pid| Process.wait(pid) }
    true
  rescue
    false
  end

  # Spawn under a PTY, drain output until the child exits, return the output.
  # Reads raise EIO on Linux when the child closes the terminal — treat as EOF.
  def run_in_pty(*cmd, chdir:, env: {}, timeout: 60)
    output = +""
    PTY.spawn(env, *cmd, chdir: chdir.to_s) do |reader, _writer, pid|
      deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + timeout
      begin
        while Process.clock_gettime(Process::CLOCK_MONOTONIC) < deadline
          ready = IO.select([reader], nil, nil, 1)
          next unless ready
          output << reader.read_nonblock(4096)
        end
        Process.kill("KILL", pid)
        raise "PTY command timed out after #{timeout}s: #{cmd.join(" ")}\n#{output}"
      rescue EOFError, Errno::EIO
        # child exited and closed the terminal
      ensure
        begin
          Process.wait(pid)
        rescue Errno::ECHILD
          # already reaped
        end
      end
    end
    output
  end
end

RSpec.configure do |config|
  config.include PtyHelper
  config.filter_run_excluding(:pty) unless PtyHelper::PTY_AVAILABLE
end
