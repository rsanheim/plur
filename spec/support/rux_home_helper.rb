module RuxHomeHelper
  module ClassMethods
    def around_with_tmp_rux_home
      around do |example|
        Dir.mktmpdir do |tmpdir|
          @tmp_rux_home = tmpdir
          ENV["RUX_HOME"] = tmpdir
          example.run
        ensure
          @tmp_rux_home = nil
          ENV.delete("RUX_HOME")
        end
      end
    end
  end

  def tmp_rux_home
    @tmp_rux_home
  end

  def rux_home
    @rux_home ||= Pathname.new(ENV["RUX_HOME"])
  end

  # Generic method to run rux with any subcommands
  # By default this will raise an error if the command fails - pass allow_error: true to allow it to fail
  #
  # @param args [Array<String>] command arguments
  # @param allow_error [Boolean] whether to allow the command to fail
  # @param env [Hash] environment variables
  # @return [TTY::Command::Result] with stdout, stderr, and exit status
  def run_rux(*args, allow_error: false, printer: :null, env: {})
    cmd = TTY::Command.new(uuid: false, printer: printer)
    if allow_error
      cmd.run!(:rux, *args, env: env)
    else
      cmd.run(:rux, *args, env: env)
    end
  end

  def run_rux_allowing_errors(*args, env: {})
    run_rux(*args, allow_error: true, env: env)
  end

  # Run rux using Open3.capture3 without argument escaping
  # Useful for testing glob expansion and shell metacharacters
  # Returns TTY::Command::Result for consistency with run_rux
  def run_rux_capture3(*args, allow_error: false, env: {})
    stdout, stderr, status = Open3.capture3(env, "rux", *args)

    result = TTY::Command::Result.new(
      stdout,
      stderr,
      status.exitstatus
    )

    # Raise if command failed and allow_error is false
    if !allow_error && status.exitstatus != 0
      raise TTY::Command::ExitError.new("rux command failed", result)
    end

    result
  end
end
