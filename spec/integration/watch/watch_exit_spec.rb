require "spec_helper"
require "pty"

# Terminals send in-band reports on stdin - focus changes, mouse moves, answers
# to queries a job asked - so those bytes land in the watch REPL's input right
# alongside what the user types.
#
# The tmux example is the real thing: a terminal that sends the reports itself
# when the user switches windows. The PTY examples cover the same ground where
# tmux isn't installed, with the harness playing the terminal.
PtyWatchTerminal = Struct.new(:reader, :writer, :screen) do
  def type(keys)
    writer.print(keys)
  end

  def wait_for(text, timeout: 10)
    deadline = Time.now + timeout
    until screen.include?(text)
      remaining = deadline - Time.now
      return false if remaining <= 0
      next unless IO.select([reader], nil, nil, [remaining, 0.2].min)

      begin
        screen << reader.read_nonblock(4096)
      rescue IO::WaitReadable
        next
      rescue EOFError, Errno::EIO
        return screen.include?(text)
      end
    end
    true
  end
end

RSpec.describe "plur watch exit" do
  include PlurWatchHelper

  context "in a real terminal", :tmux do
    it "exits on the first exit typed after the window loses and regains focus" do
      tmux_terminal(dir: default_ruby_dir) do |terminal|
        # A TUI that exited without turning focus reporting back off, which is
        # how a session ends up in this state.
        terminal.submit("printf '\\033[?1004h'")
        terminal.submit("#{plur_binary} watch run --timeout 60")
        expect(terminal.wait_for("[plur] >", timeout: 30)).to be(true), "watch never prompted:\n#{terminal.screen}"

        terminal.leave_and_return
        # The tty echoes the reports, so this confirms they landed before the
        # user typed anything - without it a slow box could pass for the wrong
        # reason.
        expect(terminal.wait_for("^[[O")).to be(true), "terminal never reported the focus change:\n#{terminal.screen}"

        terminal.submit("exit")

        expect(terminal.wait_for("Exiting watch mode...")).to be(true), "screen was:\n#{terminal.screen}"
      end
    end
  end

  context "with terminal reports on stdin", :pty do
    def watch_terminal(dir: default_ruby_dir, timeout: 20)
      PTY.spawn({"TERM" => "xterm-256color"}, plur_binary, "watch", "run", "--timeout", timeout.to_s, chdir: dir.to_s) do |reader, writer, pid|
        writer.sync = true
        terminal = PtyWatchTerminal.new(reader: reader, writer: writer, screen: +"")
        terminal.wait_for("[plur] > ")
        yield terminal
      ensure
        begin
          Process.kill("TERM", pid)
        rescue Errno::ESRCH
          # already exited
        end
      end
    end

    it "exits when the user types exit" do
      watch_terminal do |terminal|
        terminal.type("exit\n")

        expect(terminal.wait_for("Exiting watch mode...")).to be(true), "screen was:\n#{terminal.screen}"
      end
    end

    it "exits on the first exit typed after the terminal reports a focus change" do
      watch_terminal do |terminal|
        terminal.type("\e[O\e[I") # focus out, focus in - sent by the terminal, not the user
        sleep 0.2
        terminal.type("exit\n")

        expect(terminal.wait_for("Exiting watch mode...")).to be(true), "screen was:\n#{terminal.screen}"
        expect(terminal.screen).not_to include("Unknown command")
      end
    end

    it "forwards Ctrl-C to the active job" do
      with_temp_watch_project do |project|
        project.join(".plur.toml").write(<<~TOML)
          use = "rspec"

          [job.rspec]
          cmd = ["ruby", "-e", "trap('INT') { puts 'job interrupted'; exit 130 }; puts 'job started'; STDOUT.flush; sleep 60"]

          [[watch]]
          source = "spec/**/*_spec.rb"
          jobs = ["rspec"]
        TOML

        watch_terminal(dir: project) do |terminal|
          terminal.type("\n")
          expect(terminal.wait_for("job started")).to be(true), "screen was:\n#{terminal.screen}"

          terminal.type("\u0003")

          expect(terminal.wait_for("job interrupted")).to be(true), "screen was:\n#{terminal.screen}"
          expect(terminal.wait_for("Received SIGINT")).to be(true), "screen was:\n#{terminal.screen}"
        end
      end
    end
  end
end
