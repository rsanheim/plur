# Helper module for running tests in fixture projects
require 'tty-command'

module FixtureRunner
  # Run the tests in the given fixture project and return the result
  # Note that printer is set to `null` by default - this can be set to:
  # - `:null` to suppress output
  # - `:pretty` colorful output
  # - `:quiet` to show only errors
  def run_fixture_tests(project_name, printer: :null, framework:)
    project_path = project_fixture!(project_name).expand_path
    
    Bundler.with_unbundled_env do
      Dir.chdir(project_path) do
        cmd = TTY::Command.new(printer: printer)
        
        # Install dependencies if needed
        if File.exist?("Gemfile")
          cmd.run("bundle check || bundle install --quiet")
        end

        # Determine the test command based on framework
        command = case framework
        when :minitest
          test_files = Dir.glob("test/**/*_test.rb")
          ["bundle", "exec", "ruby", "-Itest"] + test_files
        when :testunit
          test_files = Dir.glob("test/**/*_test.rb")
          ["bundle", "exec", "ruby", "-Itest"] + test_files
        when :rspec
          ["bundle", "exec", "rspec"]
        else
          raise "Cannot determine test framework"
        end

        # Run the tests and capture output
        puts "Running command in #{Dir.pwd}: #{command.join(' ')}" if ENV['DEBUG']
        
        cmd.run!(*command)
      end
    end
  end

  # Helper to run tests and compare with rux output
  def compare_with_native_runner(project_name, framework: :auto)
    native_result = run_fixture_tests(project_name, framework: framework)
    
    # Return the native result for comparison
    native_result
  end

  # Extract test counts from output
  def extract_test_stats(output, framework)
    case framework
    when :minitest
      if output =~ /(\d+) runs, (\d+) assertions, (\d+) failures, (\d+) errors, (\d+) skips/
        {
          runs: $1.to_i,
          assertions: $2.to_i,
          failures: $3.to_i,
          errors: $4.to_i,
          skips: $5.to_i
        }
      end
    when :testunit
      if output =~ /(\d+) tests, (\d+) assertions, (\d+) failures, (\d+) errors, (\d+) pendings, (\d+) omissions/
        {
          tests: $1.to_i,
          assertions: $2.to_i,
          failures: $3.to_i,
          errors: $4.to_i,
          pendings: $5.to_i,
          omissions: $6.to_i
        }
      end
    when :rspec
      if output =~ /(\d+) examples?, (\d+) failures?/
        {
          examples: $1.to_i,
          failures: $2.to_i
        }
      end
    end
  end
end

# Make it available in RSpec
RSpec.configure do |config|
  config.include FixtureRunner
end