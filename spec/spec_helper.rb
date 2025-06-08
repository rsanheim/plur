require "fileutils"
require "open3"
require "ostruct"
require "pathname"
require "stringio"
require "super_diff/rspec"
require "timecop"
require "tmpdir"
require "tty-command"

require "backspin"

ROOT_PATH = Pathname.new(__dir__).parent
DEFAULT_RUBY_DIR = ROOT_PATH.join("fixtures", "projects", "default-ruby")
# The default 'run time' for rux watch for integration tests
# After this time rux watch will automatically exit
DEFAULT_RUX_WATCH_TIMEOUT = 2

# Load all support files
Dir[File.join(__dir__, "support", "**", "*.rb")].each { |f| require f }

RSpec.configure do |config|
  def chdir(path)
    Dir.chdir(path) do
      yield
    end
  end

  def rux_binary
    @rux_binary ||= File.join(__dir__, "..", "rux", "rux")
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

  config.filter_run_when_matching :focus
  config.disable_monkey_patching!

  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups

  # Restore default-ruby state after the entire test suite
  config.after(:suite) do
    Dir.chdir(DEFAULT_RUBY_DIR) do
      # Reset any file changes made during tests
      system("git checkout -- .", out: File::NULL, err: File::NULL)
    end
  end
end
