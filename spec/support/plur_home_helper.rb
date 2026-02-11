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

  # Temporarily remove plur parallel env vars that leak from the outer
  # test runner when the suite itself is run via plur.
  def without_plur_parallel_env
    saved = {}
    %w[TEST_ENV_NUMBER PARALLEL_TEST_GROUPS].each do |key|
      saved[key] = ENV.delete(key)
    end
    yield
  ensure
    saved.each { |key, value| ENV[key] = value if value }
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

  def run_plur_allowing_errors(*args, printer: :null, env: {})
    run_plur(*args, allow_error: true, printer: printer, env: env)
  end
end
