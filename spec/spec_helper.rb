require "fileutils"
require "json"
require "open3"
require "ostruct"
require "stringio"
require "timecop"
require "tmpdir"
require "tty-command"
require "backspin"

# The suite shells out to the `plur` binary, which on dev machines is usually a
# mise shim. Several integration specs spawn it with an isolated HOME, so mise
# can't reach its trust store and aborts with "config not trusted", polluting
# the subprocess stderr. Our mise configs are trusted, so auto-confirm instead
# of failing. No effect when `plur` is a plain binary, since mise never runs.
ENV["MISE_YES"] ||= "1"

ROOT_PATH = Pathname.new(__dir__).parent
DEFAULT_RUBY_SPEC_FILE_COUNT = 13

Backspin.configure do |config|
  config.backspin_dir = ROOT_PATH.join("fixtures", "backspin")
end

Dir[File.join(__dir__, "support", "**", "*.rb")].each { |f| require f }

RSpec.configure do |config|
  config.include PlurHomeHelper
  config.extend PlurHomeHelper::ClassMethods

  config.filter_run_when_matching :focus
  config.disable_monkey_patching!
  config.filter_run_excluding :skip_if_ci if ENV["CI"]
  config.shared_context_metadata_behavior = :apply_to_host_groups

  # Scrub plur's parallel env vars before each spec so the outer test
  # runner (which may itself be plur) doesn't leak into child processes.
  config.around do |example|
    without_plur_parallel_env { example.run }
  end

  def chdir(path)
    Dir.chdir(path) do
      yield
    end
  end

  # Matches any ANSI color escape sequence in command output.
  def ansi
    /\e\[\d+m/
  end

  # For golden comparisons over a pipe: rspec prints a progress line there, plur does not.
  def without_progress_line(output)
    output.sub(/\A(?:(?:\e\[3\dm)?[.F*E](?:\e\[0m)?)+\n/, "")
  end

  def default_ruby_dir
    @default_ruby_dir ||= project_fixture("default-ruby")
  end

  def default_rails_dir
    @default_rails_dir ||= project_fixture("default-rails")
  end

  def project_fixture(name)
    Pathname.new(__dir__).parent.join("fixtures", "projects", name)
  end

  def run_rspec(file_or_glob, *args)
    cmd_array = %W[bundle exec rspec #{file_or_glob}]
    cmd_array += args if args.any?
    Open3.capture3(*cmd_array)
  end

  def project_fixture!(name)
    project_fixture(name).tap do |path|
      unless path.exist?
        raise "Project fixture does not exist for #{name} at path: #{path}"
      end
    end
  end

  # Yields a private copy of a fixture project. Specs that write files must
  # use one: other plur workers run the shared fixtures concurrently.
  def with_fixture_copy(source)
    FileUtils.mkdir_p(ROOT_PATH.join("tmp"))

    Dir.mktmpdir("plur-fixture", ROOT_PATH.join("tmp").to_s) do |tmpdir|
      project_dir = Pathname.new(tmpdir).join("project")
      FileUtils.cp_r(source, project_dir)
      yield project_dir
    end
  end

  # A file written into a shared fixture's spec/ or test/ tree, even briefly,
  # races every worker discovering tests there. Fail the run if any changed.
  fixture_test_dirs = -> { Dir.glob(ROOT_PATH.join("fixtures/projects/*/{spec,test}/**/").to_s).to_h { |dir| [dir, File.mtime(dir)] } }
  fixture_test_dirs_before = nil
  config.before(:suite) { fixture_test_dirs_before = fixture_test_dirs.call }
  config.after(:suite) do
    changed = fixture_test_dirs.call.reject { |dir, mtime| fixture_test_dirs_before[dir] == mtime }
    raise "Specs modified shared fixture dirs (use with_fixture_copy): #{changed.keys.join(", ")}" if changed.any?
  end

  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end
end
