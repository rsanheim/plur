require "pty"
require "tty-command"

# Runs a command under a real pseudo-terminal so specs can assert TTY-side
# behavior (plur's auto color mode). Pipe-side behavior needs no PTY — the
# regular run_plur helper already runs through pipes.
module PtyHelper
  PTY_AVAILABLE = begin
    PTY.open { true }
  rescue
    false
  end

  # tty-command silently falls back to pipes when a PTY is unavailable, so
  # :pty specs are excluded (below) rather than left to fail on the fallback.
  def run_in_pty(*cmd, chdir:, env: {}, timeout: 60)
    result = TTY::Command.new(uuid: false, printer: :null)
      .run!(*cmd, env: env, chdir: chdir.to_s, pty: true, timeout: timeout)
    result.out + result.err
  end
end

RSpec.configure do |config|
  config.include PtyHelper
  config.filter_run_excluding(:pty) unless PtyHelper::PTY_AVAILABLE
end
