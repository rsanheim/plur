require "open3"
require "timeout"

module PlurWatchHelper
  DEFAULT_PLUR_WATCH_TIMEOUT = ENV["CI"] ? 12 : 5
  READY_SIGNAL = "s/self/live@"
  READY_SETTLE_SECONDS = 0.3

  WatchResult = Struct.new(:out, :err, :exit_status, keyword_init: true) do
    def success?
      return exit_status.success? if exit_status.respond_to?(:success?)
      exit_status == 0
    end
  end

  # Runs plur watch via Open3 with process lifecycle control.
  #
  # When a block is given, it fires after the watcher emits its ready signal.
  # When until_output is set, the process is killed as soon as that string
  # appears in stdout or stderr (early exit for faster tests).
  #
  # @param dir [Pathname, String] directory to run in
  # @param timeout [Integer] plur --timeout value in seconds
  # @param debounce [Integer] debounce delay in milliseconds
  # @param env [Hash] environment variables
  # @param until_output [String, nil] kill process when this string appears in output
  # @return [WatchResult]
  def run_plur_watch(dir: default_ruby_dir, timeout: DEFAULT_PLUR_WATCH_TIMEOUT,
    debounce: nil, env: {}, until_output: nil, &block)
    Dir.chdir(dir) do
      cmd_args = [plur_binary, "--debug", "watch", "run", "--timeout", timeout.to_s]
      cmd_args += ["--debounce", debounce.to_s] if debounce

      out = +""
      err = +""

      Open3.popen3(env, *cmd_args) do |stdin, stdout, stderr, wait_thr|
        pid = wait_thr.pid
        stdin.close

        deadline = Time.now + timeout + 5
        block_fired = false
        streams = [stdout, stderr]
        ready_state = {count: 0, stable_since: nil}

        while !streams.empty? && Time.now < deadline
          ready = IO.select(streams, nil, nil, 0.1)
          if ready
            ready[0].each do |io|
              data = io.read_nonblock(16384)
              (io == stdout) ? out << data : err << data
            rescue IO::WaitReadable
              next
            rescue EOFError
              streams.delete(io)
            end
          end

          # After ready signal, fire the block once
          if block_given? && !block_fired && watch_ready?(err, ready_state)
            yield
            block_fired = true
          end

          # Early exit when acceptance criteria met
          if until_output && (err.include?(until_output) || out.include?(until_output))
            begin
              Process.kill("TERM", pid)
            rescue Errno::ESRCH
              # process already exited
            end
            break
          end
        end

        # Wait for graceful shutdown (plur handles SIGTERM cleanly)
        begin
          Timeout.timeout(3) { wait_thr.value }
        rescue Timeout::Error
          begin
            Process.kill("KILL", pid)
          rescue Errno::ESRCH
            # process already exited
          end
          wait_thr.value
        end

        # Drain remaining buffered output after process exit
        [stdout, stderr].each do |io|
          next if io.closed?
          loop do
            data = io.read_nonblock(16384)
            (io == stdout) ? out << data : err << data
          rescue IOError, IO::WaitReadable
            break
          end
        end

        WatchResult.new(out: out, err: err, exit_status: wait_thr.value)
      end
    end
  end

  def run_plur_watch_interactive(dir: default_ruby_dir, timeout: DEFAULT_PLUR_WATCH_TIMEOUT,
    debounce: nil, env: {}, commands:, command_delay: 0)
    Dir.chdir(dir) do
      cmd_args = [plur_binary, "--debug", "watch", "run", "--timeout", timeout.to_s]
      cmd_args += ["--debounce", debounce.to_s] if debounce

      out = +""
      err = +""

      Open3.popen3(env, *cmd_args) do |stdin, stdout, stderr, wait_thr|
        deadline = Time.now + timeout + 5
        streams = [stdout, stderr]
        commands_sent = false
        ready_state = {count: 0, stable_since: nil}

        while !streams.empty? && Time.now < deadline
          ready = IO.select(streams, nil, nil, 0.1)
          if ready
            ready[0].each do |io|
              data = io.read_nonblock(16384)
              (io == stdout) ? out << data : err << data
            rescue IO::WaitReadable
              next
            rescue EOFError
              streams.delete(io)
            end
          end

          next unless !commands_sent && watch_ready?(err, ready_state)

          commands.each_with_index do |command, index|
            stdin.puts(command)
            sleep command_delay if command_delay > 0 && index < commands.length - 1
          end
          stdin.close
          commands_sent = true
        end

        begin
          Timeout.timeout(3) { wait_thr.value }
        rescue Timeout::Error
          begin
            Process.kill("KILL", wait_thr.pid)
          rescue Errno::ESRCH
            # process already exited
          end
          wait_thr.value
        end

        [stdout, stderr].each do |io|
          next if io.closed?
          loop do
            data = io.read_nonblock(16384)
            (io == stdout) ? out << data : err << data
          rescue IOError, IO::WaitReadable
            break
          end
        end

        WatchResult.new(out: out, err: err, exit_status: wait_thr.value)
      end
    end
  end

  # Helper to check for file change events in the new log format
  # Note: Logger quotes string values, so we match the quoted format
  def expect_file_change_logged(output, file_path)
    expect(output).to include('event="modify" type="file"')
    expect(output).to include("path=\"#{file_path}\"")
  end

  # Helper to check for spec run events in the new log format
  def expect_spec_run_logged(output, spec_path)
    expect(output).to include("event=\"run_command\" path=\"#{spec_path}\"")
  end

  def watch_ready?(stderr_output, ready_state)
    ready_count = stderr_output.scan(READY_SIGNAL).count
    return false if ready_count == 0

    if ready_count != ready_state[:count]
      ready_state[:count] = ready_count
      ready_state[:stable_since] = Time.now
      return false
    end

    return false if ready_state[:stable_since].nil?

    Time.now - ready_state[:stable_since] >= READY_SETTLE_SECONDS
  end
end
