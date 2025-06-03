require "bundler/setup"
require "backspin"
require "timecop"

RSpec.configure do |config|
  # Enable flags like --only-failures and --next-failure
  config.example_status_persistence_file_path = ".rspec_status"

  # Disable RSpec exposing methods globally on `Module` and `main`
  config.disable_monkey_patching!

  config.expect_with :rspec do |c|
    c.syntax = :expect
  end

  # Clean up dubplates after each test
  config.after(:each) do
    dubplate_dir = File.join(Dir.pwd, "tmp", "backspin")
    FileUtils.rm_rf(dubplate_dir) if Dir.exist?(dubplate_dir)
  end
end
