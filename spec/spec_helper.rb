require "fileutils"
require "open3"
require "pathname"
require "super_diff/rspec"
require "tmpdir"
require "tty-command"
require "stringio"
require "ostruct"

BACKSPIN_PATH = Pathname.new(__dir__).parent.join("vendor", "backspin", "lib").expand_path.freeze
$LOAD_PATH.unshift(BACKSPIN_PATH)

require "backspin"

ROOT_PATH = Pathname.new(__dir__).parent
RUX_RUBY_DIR = ROOT_PATH.join("test_fixtures", "rux-ruby")
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

  def rux_ruby_dir
    @rux_ruby_dir ||= Pathname.new(__dir__).parent.join("test_fixtures", "rux-ruby")
  end

  def test_app_dir
    @test_app_dir ||= Pathname.new(__dir__).parent.join("test_fixtures", "test_app")
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

  # Restore rux-ruby state after the entire test suite
  config.after(:suite) do
    Dir.chdir(RUX_RUBY_DIR) do
      # Reset any file changes made during tests
      system("git checkout -- .", out: File::NULL, err: File::NULL)
    end
  end
end
