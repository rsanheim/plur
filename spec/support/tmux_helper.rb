require "pty"
require "securerandom"

# Drives a real terminal so specs can act as a user: real keystrokes, real
# terminal behavior, and assertions against what is actually rendered on
# screen. tmux is the terminal emulator; each session runs on its own socket so
# specs never touch a developer's own tmux sessions.
#
# The session has two windows and an attached client. Both matter: tmux only
# delivers focus reports to a pane when a client is attached, and switching
# windows is what makes it send them.
module TmuxHelper
  TMUX_AVAILABLE = system("tmux", "-V", out: File::NULL, err: File::NULL)
  SHELL = "bash --noprofile --norc".freeze
  WORK_PANE = "work:0.0".freeze

  class Terminal
    def initialize(socket)
      @socket = socket
    end

    # What is on screen, as rendered - escape sequences are interpreted by tmux
    # the same way they would be by any other terminal.
    def screen
      `tmux -L #{@socket} capture-pane -p -t #{WORK_PANE}`
    end

    def type(text)
      tmux("send-keys", "-t", WORK_PANE, "-l", text)
    end

    def submit(line)
      type(line)
      tmux("send-keys", "-t", WORK_PANE, "Enter")
    end

    def wait_for(text, timeout: 15)
      deadline = Time.now + timeout
      until screen.include?(text)
        return false if Time.now > deadline
        sleep 0.2
      end
      true
    end

    # Switches to the other window and back. The terminal sends focus out and
    # focus in reports to the program running here, exactly as it does when a
    # user switches away to another window and comes back.
    def leave_and_return
      tmux("select-window", "-t", "work:1")
      sleep 0.5
      tmux("select-window", "-t", "work:0")
      sleep 0.5
    end

    private

    def tmux(*args)
      system("tmux", "-L", @socket, *args, out: File::NULL, err: File::NULL)
    end
  end

  def tmux_terminal(dir:, width: 200, height: 50)
    socket = "plur-spec-#{Process.pid}-#{SecureRandom.hex(4)}"
    tmux_control(socket, "new-session", "-d", "-s", "work", "-x", width.to_s, "-y", height.to_s, "-c", dir.to_s, SHELL)
    tmux_control(socket, "set-option", "-g", "focus-events", "on")
    tmux_control(socket, "new-window", "-t", "work", "-c", dir.to_s, SHELL)
    tmux_control(socket, "select-window", "-t", "work:0")

    reader, _writer, client_pid = PTY.spawn({"TERM" => "xterm-256color"}, "tmux", "-L", socket, "attach", "-t", "work")
    drain = Thread.new do
      loop { reader.readpartial(4096) }
    rescue IOError, Errno::EIO
      nil
    end
    wait_for_tmux_client(socket)

    yield Terminal.new(socket)
  ensure
    drain&.kill
    tmux_control(socket, "kill-server")
    begin
      Process.kill("TERM", client_pid) if client_pid
    rescue Errno::ESRCH
      # client already gone
    end
  end

  def tmux_control(socket, *args)
    system("tmux", "-L", socket, *args, out: File::NULL, err: File::NULL)
  end

  # Focus reports depend on an attached client, so wait for one before typing.
  def wait_for_tmux_client(socket, timeout: 10)
    deadline = Time.now + timeout
    until `tmux -L #{socket} list-clients 2>/dev/null`.strip != ""
      raise "no tmux client attached after #{timeout}s" if Time.now > deadline
      sleep 0.1
    end
  end
end

RSpec.configure do |config|
  config.include TmuxHelper
  config.filter_run_excluding(:tmux) unless TmuxHelper::TMUX_AVAILABLE
end
