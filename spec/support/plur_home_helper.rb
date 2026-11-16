module PlurHomeHelper
  DEFAULT_PLUR_BINARY = begin
    gobin = `go env GOBIN`.strip
    gobin = File.join(`go env GOPATH`.strip, "bin") if gobin.empty?
    File.join(gobin, "plur")
  end

  # Outer-environment settings that would change how a spawned plur behaves: its
  # home and config file, debug logging, parallel vars when the suite itself
  # runs via plur, and color conventions a developer machine or CI may export.
  # Specs that exercise these pass them explicitly via run_plur's env: param.
  OUTER_PLUR_ENV = %w[
    PLUR_HOME PLUR_CONFIG_FILE PLUR_DEBUG PLUR_WORKERS PARALLEL_TEST_PROCESSORS TEST_ENV_NUMBER
    PARALLEL_TEST_GROUPS NO_COLOR PLUR_COLOR PLUR_FORMATTER PLUR_DEV_PROFILE
  ].freeze

  # Every example runs inside this. Spawned processes get a new, empty HOME and
  # none of OUTER_PLUR_ENV, so whatever they keep under HOME (plur's ~/.plur and
  # ~/.plur.toml, tool dotfiles) starts clean and never touches the home of
  # whoever runs the suite.
  def with_isolated_env
    saved = ENV.to_h
    Dir.mktmpdir("plur-spec-home-") do |home|
      @isolated_home = Pathname.new(home)
      isolate_env(home)
      yield
    end
  ensure
    ENV.replace(saved)
  end

  # Use instead of Bundler.with_unbundled_env, which resets ENV to what it was
  # when Bundler loaded and so would hand the block the real HOME.
  def with_unbundled_env
    Bundler.with_unbundled_env do
      yield
    end
  end

  def isolate_env(home)
    ENV["HOME"] = home
    OUTER_PLUR_ENV.each { |key| ENV.delete(key) }
  end

  # Specs build paths from this rather than ENV["HOME"], so code that runs
  # outside an example can never write to or delete from the real home.
  def isolated_home
    @isolated_home || raise("isolated_home is only available inside an example")
  end

  def plur_home
    isolated_home.join(".plur")
  end

  def plur_binary
    ENV.fetch("PLUR_BINARY", DEFAULT_PLUR_BINARY)
  end

  # Generic method to run plur with any subcommands
  # By default this will raise an error if the command fails - pass allow_error: true to allow it to fail
  #
  # @param args [Array<String>] command arguments
  # @param allow_error [Boolean] whether to allow the command to fail
  # @param env [Hash] environment variables
  # @return [TTY::Command::Result] with stdout, stderr, and exit status
  def run_plur(*args, allow_error: false, printer: :null, env: {})
    plur_cmd = plur_binary
    cmd = TTY::Command.new(uuid: false, printer: printer)
    if allow_error
      cmd.run!(plur_cmd, *args, env: env)
    else
      cmd.run(plur_cmd, *args, env: env)
    end
  end

  def run_plur_allowing_errors(*args, printer: :null, env: {})
    run_plur(*args, allow_error: true, printer: printer, env: env)
  end
end
