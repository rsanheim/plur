require "open3"
require "timeout"

module PlurWatchHelper
  DEFAULT_PLUR_WATCH_TIMEOUT = ENV["CI"] ? 12 : 5
  READY_SIGNAL = "s/self/live@"
  READY_SIGNAL_PATTERN = /#{Regexp.escape(READY_SIGNAL)}([^"\s]+)/
  WATCH_DIRS_PATTERN = /Watch directories after filtering dirs=\[([^\]]*)\]/
  READY_SETTLE_SECONDS = 0.3

  WatchResult = Struct.new(:out, :err, :exit_status, keyword_init: true) do
    def success?
      return exit_status.success? if exit_status.respond_to?(:success?)
      exit_status == 0
    end
  end

  WatchProcess = Struct.new(:stdin, :out, :err, :ready_state, keyword_init: true) do
    def close_stdin
      stdin.close unless stdin.closed?
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
  # @param ready_dirs [Array<String>, Symbol, nil] watcher directories to wait for, or :detected to infer from startup logs
  # @return [WatchResult]
  def run_plur_watch(dir: default_ruby_dir, timeout: DEFAULT_PLUR_WATCH_TIMEOUT,
    debounce: nil, env: {}, until_output: nil, ready_dirs: :detected, &block)
    block_fired = false

    capture_plur_watch_process(dir: dir, timeout: timeout, debounce: debounce, env: env) do |process|
      process.close_stdin

      if block_given? && !block_fired && watch_ready?(process.err, process.ready_state, ready_dirs: ready_dirs)
        yield
        block_fired = true
      end

      if until_output && (process.err.include?(until_output) || process.out.include?(until_output))
        :terminate
      end
    end
  end

  def run_plur_watch_interactive(commands:, dir: default_ruby_dir, timeout: DEFAULT_PLUR_WATCH_TIMEOUT,
    debounce: nil, env: {}, command_delay: 0, ready_dirs: :detected)
    commands_sent = false

    capture_plur_watch_process(dir: dir, timeout: timeout, debounce: debounce, env: env) do |process|
      next unless !commands_sent && watch_ready?(process.err, process.ready_state, ready_dirs: ready_dirs)

      commands.each_with_index do |command, index|
        process.stdin.puts(command)
        sleep command_delay if command_delay > 0 && index < commands.length - 1
      end
      process.close_stdin
      commands_sent = true
    end
  end

  def with_temp_watch_project(source: default_ruby_dir)
    FileUtils.mkdir_p(ROOT_PATH.join("tmp"))

    Dir.mktmpdir("plur-watch-project", ROOT_PATH.join("tmp").to_s) do |tmpdir|
      project_dir = Pathname.new(tmpdir).join("project")
      FileUtils.cp_r(source, project_dir)
      yield project_dir
    end
  end

  def capture_plur_watch_process(dir:, timeout:, debounce: nil, env: {})
    Dir.chdir(dir) do
      cmd_args = [plur_binary, "--debug", "watch", "run", "--timeout", timeout.to_s]
      cmd_args += ["--debounce", debounce.to_s] if debounce

      out = +""
      err = +""

      Open3.popen3(env, *cmd_args) do |stdin, stdout, stderr, wait_thr|
        deadline = Time.now + timeout + 5
        streams = [stdout, stderr]
        ready_state = {count: 0, stable_since: nil}
        process = WatchProcess.new(stdin: stdin, out: out, err: err, ready_state: ready_state)

        while !streams.empty? && Time.now < deadline
          read_available_watch_output(streams, stdout, out, err)
          action = yield process if block_given?

          if action == :terminate
            terminate_process(wait_thr.pid, "TERM")
            break
          end
        end

        exit_status = wait_for_watch_process(wait_thr)
        drain_watch_output(stdout, stderr, out, err)

        WatchResult.new(out: out, err: err, exit_status: exit_status)
      end
    end
  end

  def read_available_watch_output(streams, stdout, out, err)
    ready = IO.select(streams, nil, nil, 0.1)
    return unless ready

    ready[0].each do |io|
      data = io.read_nonblock(16384)
      (io == stdout) ? out << data : err << data
    rescue IO::WaitReadable
      next
    rescue EOFError
      streams.delete(io)
    end
  end

  def wait_for_watch_process(wait_thr)
    Timeout.timeout(3) { wait_thr.value }
  rescue Timeout::Error
    terminate_process(wait_thr.pid, "KILL")
    wait_thr.value
  end

  def drain_watch_output(stdout, stderr, out, err)
    [stdout, stderr].each do |io|
      next if io.closed?
      loop do
        data = io.read_nonblock(16384)
        (io == stdout) ? out << data : err << data
      rescue IOError, IO::WaitReadable
        break
      end
    end
  end

  def terminate_process(pid, signal)
    Process.kill(signal, pid)
  rescue Errno::ESRCH
    # process already exited
  end

  # Helper to check for file change events in the new log format
  # Note: Logger quotes string values, so we match the quoted format
  def expect_file_change_logged(output, file_path)
    expect(output).to include('event="modify" type="file"')
    expect(output).to include("path=\"#{file_path}\"")
  end

  def watch_ready?(stderr_output, ready_state, ready_dirs: nil)
    ready_dirs = detected_ready_dirs(stderr_output) if ready_dirs == :detected
    ready_count = ready_dirs ? live_ready_dir_count(stderr_output, ready_dirs) : stderr_output.scan(READY_SIGNAL).count
    return false if ready_count == 0
    return false if ready_dirs && ready_count < ready_dirs.count

    if ready_count != ready_state[:count]
      ready_state[:count] = ready_count
      ready_state[:stable_since] = Time.now
      return false
    end

    return false if ready_state[:stable_since].nil?

    Time.now - ready_state[:stable_since] >= READY_SETTLE_SECONDS
  end

  def detected_ready_dirs(stderr_output)
    raw_dirs = stderr_output[WATCH_DIRS_PATTERN, 1]
    return [] unless raw_dirs

    raw_dirs.split(" ").reject(&:empty?)
  end

  def live_ready_dir_count(stderr_output, ready_dirs)
    live_paths = stderr_output.scan(READY_SIGNAL_PATTERN).flatten

    ready_dirs.count do |dir|
      live_paths.any? { |path| live_ready_dir_matches?(path, dir) }
    end
  end

  def live_ready_dir_matches?(path, dir)
    return true if dir == "."

    path == dir || path.end_with?("/#{dir}")
  end
end
