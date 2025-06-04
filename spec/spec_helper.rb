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
# The default 'run time' for rux watch for integration tests
# After this time rux watch will automatically exit
DEFAULT_RUX_WATCH_TIMEOUT = 2

# Shared module for watch spec helpers
module WatchSpecHelpers
  def run_rux_watch(dir: rux_ruby_dir, timeout: DEFAULT_RUX_WATCH_TIMEOUT, debounce: nil, env: {}, &block)
    Dir.chdir(dir) do
      cmd = "#{rux_binary} watch --timeout #{timeout}"
      cmd += " --debounce #{debounce}" if debounce

      # Add GO_LOG=debug to env if not already set for better debugging
      full_env = {"GO_LOG" => "debug"}.merge(env)
      
      if block_given?
        # Run in background mode for file modification tests
        stdout, stderr, status = nil
        
        thread = Thread.new do
          stdout, stderr, status = Open3.capture3(full_env, cmd)
        end
        
        # Give watch time to start
        sleep 0.5
        
        # Perform actions while watch is running
        yield
        
        # Wait for thread to complete (timeout will exit it)
        thread.join
        
        OpenStruct.new(
          out: stdout,
          err: stderr,
          success?: status&.success?
        )
      else
        # Simple synchronous execution
        stdout, stderr, status = Open3.capture3(full_env, cmd)
        
        OpenStruct.new(
          out: stdout,
          err: stderr,
          success?: status.success?
        )
      end
    end
  end
end

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
    @rux_ruby_dir ||= Pathname.new(__dir__).parent.join("rux-ruby")
  end

  config.include WatchSpecHelpers

  config.filter_run_when_matching :focus
  config.disable_monkey_patching!

  config.expect_with :rspec do |expectations|
    expectations.include_chain_clauses_in_custom_matcher_descriptions = true
  end

  config.mock_with :rspec do |mocks|
    mocks.verify_partial_doubles = true
  end

  config.shared_context_metadata_behavior = :apply_to_host_groups
end