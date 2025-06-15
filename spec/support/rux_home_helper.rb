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
  def run_rux(*args, allow_error: false, env: {})
    cmd = TTY::Command.new(uuid: false, printer: :null)
    if allow_error
      cmd.run!(:rux, *args, env: env)
    else
      cmd.run(:rux, *args, env: env)
    end
  end

end