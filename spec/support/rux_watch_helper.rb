require "tty-command"
require "open3"

module RuxWatchHelper
  DEFAULT_RUX_WATCH_TIMEOUT = ENV["CI"] ? 8 : 3

  # Runs rux watch with TTY::Command and returns a proper result object
  # @param dir [String] directory to run in
  # @param timeout [Integer] timeout in seconds
  # @param debounce [Integer] debounce delay in milliseconds
  # @param env [Hash] environment variables
  # @param block [Proc] optional block to execute while watch is running
  # @return [TTY::Command::Result] with stdout, stderr, and exit status
  def run_rux_watch(dir: default_ruby_dir, timeout: DEFAULT_RUX_WATCH_TIMEOUT, debounce: nil, env: {}, &block)
    Dir.chdir(dir) do
      cmd_args = ["rux", "--debug", "watch", "run", "--timeout", timeout.to_s]
      cmd_args += ["--debounce", debounce.to_s] if debounce

      full_env = env

      cmd = TTY::Command.new(uuid: false, printer: :null)

      if block_given?
        # Run command asynchronously with block
        result = nil
        thread = Thread.new do
          result = cmd.run(cmd_args.join(" "), env: full_env)
        end

        # Give the watcher time to start
        sleep 0.5

        # Execute the block
        yield

        # Wait for the command to finish
        thread.join

        # Return the result
        result
      else
        # Run command synchronously
        cmd.run!(cmd_args.join(" "), env: full_env)
      end
    end
  end

  # Captures watch output with streaming support for tests that need to
  # react to output in real-time
  def capture_watch_output(rux_timeout: DEFAULT_RUX_WATCH_TIMEOUT, debounce: nil, &block)
    Dir.chdir(default_ruby_dir) do
      args = "rux --debug watch run --timeout #{rux_timeout}"
      args += " --debounce #{debounce}" if debounce

      env = {}
      # TTY::Command timeout needs to be longer than rux watch timeout to avoid timeout errors
      full_timeout = rux_timeout + 2
      cmd = TTY::Command.new(timeout: full_timeout, uuid: false, pty: true, printer: :null)

      streamed_out, streamed_err = [], []

      result = cmd.run!(args, env: env) do |out, err|
        streamed_out << out
        streamed_err << err
        block.call(out, err) if block_given?
      end

      # Return the result object plus the streamed arrays
      [result, streamed_out, streamed_err]
    end
  end

  # Helper to check for file change events in the new log format
  def expect_file_change_logged(output, file_path)
    expect(output).to include("watch event=modify type=file")
    expect(output).to include("path=#{file_path}")
  end

  # Helper to check for spec run events in the new log format
  def expect_spec_run_logged(output, spec_path)
    expect(output).to include("rux event=run_spec path=#{spec_path}")
  end
end
