require "fileutils"
require "open3"
require "pathname"
require "super_diff/rspec"
require "tmpdir"

ROOT_PATH = Pathname.new(__dir__).parent

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
