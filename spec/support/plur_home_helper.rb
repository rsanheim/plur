module PlurHomeHelper
  module ClassMethods
    def around_with_tmp_plur_home
      around do |example|
        Dir.mktmpdir do |tmpdir|
          @tmp_plur_home = tmpdir
          ENV["PLUR_HOME"] = tmpdir
          example.run
        ensure
          @tmp_plur_home = nil
          ENV.delete("PLUR_HOME")
        end
      end
    end
  end

  def tmp_plur_home
    @tmp_plur_home
  end

  def plur_home
    @plur_home ||= Pathname.new(ENV["PLUR_HOME"])
  end

  # Generic method to run plur with any subcommands
  # By default this will raise an error if the command fails - pass allow_error: true to allow it to fail
  #
  # @param args [Array<String>] command arguments
  # @param allow_error [Boolean] whether to allow the command to fail
  # @param env [Hash] environment variables
  # @return [TTY::Command::Result] with stdout, stderr, and exit status
  def run_plur(*args, allow_error: false, printer: :null, env: {})
    cmd = TTY::Command.new(uuid: false, printer: printer)
    if allow_error
      cmd.run!(:plur, *args, env: env)
    else
      cmd.run(:plur, *args, env: env)
    end
  end

  def run_plur_allowing_errors(*args, env: {})
    run_plur(*args, allow_error: true, env: env)
  end

  # Run plur using Open3.capture3 without argument escaping
  # Useful for testing glob expansion and shell metacharacters
  # Returns TTY::Command::Result for consistency with run_plur
  def run_plur_capture3(*args, allow_error: false, env: {})
    stdout, stderr, status = Open3.capture3(env, "plur", *args)

    result = TTY::Command::Result.new(
      stdout,
      stderr,
      status.exitstatus
    )

    # Raise if command failed and allow_error is false
    if !allow_error && status.exitstatus != 0
      raise TTY::Command::ExitError.new("plur command failed", result)
    end

    result
  end
end
