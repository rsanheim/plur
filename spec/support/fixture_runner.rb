# Helper module for running tests in fixture projects
require "open3"
require "tty-command"

module FixtureRunner
  # Run the tests in the given fixture project and return the result
  # Note that printer is set to `null` by default - this can be set to:
  # - `:null` to suppress output
  # - `:pretty` colorful output
  # - `:quiet` to show only errors
  def run_fixture_tests(project_name, framework:, printer: :null)
    project_path = project_fixture!(project_name).expand_path

    Bundler.with_unbundled_env do
      Dir.chdir(project_path) do
        # Install dependencies if needed
        if File.exist?("Gemfile")
          _, _, status = Open3.capture3("bundle check")
          unless status.success?
            _, _, _ = Open3.capture3("bundle install --quiet")
          end
        end

        # Determine the test command based on framework
        command = case framework
        when :minitest, :testunit
          test_files = Dir.glob("test/**/*_test.rb")
          if test_files.empty?
            raise "No test files found in #{project_path}/test"
          end

          # Build the inline script like parallel_tests does
          require_list = test_files.map { |f| f.gsub(" ", "\\ ") }.join(" ")
          [
            "bundle", "exec", "ruby",
            "-Itest",
            "-e", "%w[#{require_list}].each { |f| require %{./\#{f}} }"
          ]
        when :rspec
          ["bundle", "exec", "rspec"]
        else
          raise "Cannot determine test framework"
        end

        # Run the tests and capture output
        puts "Running command in #{Dir.pwd}: #{command.join(" ")}" if ENV["DEBUG"]

        stdout, stderr, status = Open3.capture3(*command)

        # Return a TTY::Command::Result directly
        TTY::Command::Result.new(status.exitstatus, stdout, stderr)
      end
    end
  end

  # Helper to run tests and compare with plur output
  def compare_with_native_runner(project_name, framework: :auto)
    run_fixture_tests(project_name, framework: framework)

    # Return the native result for comparison
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
