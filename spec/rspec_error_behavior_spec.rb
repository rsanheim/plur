require "spec_helper"
require "open3"

RSpec.describe "RSpec error behavior" do
  let(:fixture_dir) { project_fixture("rspec-errors") }

  context "running RSpec directly" do
    it "shows behavior for passing tests" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/passing_spec.rb")
        
        puts "\n=== Passing tests ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?/).first&.strip}"
        puts "STDERR empty: #{stderr.empty?}"
        
        expect(status.exitstatus).to eq(0)
        expect(stdout).to include("2 examples, 0 failures")
      end
    end

    it "shows behavior for test failures" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/test_failure_spec.rb")
        
        puts "\n=== Test failures ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?/).first&.strip}"
        puts "STDERR empty: #{stderr.empty?}"
        
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include("2 examples, 1 failure")
      end
    end

    it "shows behavior for unhandled exceptions" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/exception_spec.rb")
        
        puts "\n=== Unhandled exceptions ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?/).first&.strip}"
        puts "STDERR empty: #{stderr.empty?}"
        
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include("2 examples, 1 failure")
      end
    end

    it "shows behavior for syntax errors in spec" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/syntax_error_spec.rb")
        
        puts "\n=== Syntax error in spec ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?.*error/).first&.strip}"
        puts "Error message in STDOUT: #{stdout.include?('SyntaxError')}"
        puts "STDERR: #{stderr.inspect}"
        
        # Show first few lines of error
        puts "Error details:"
        stdout.lines.take(10).each { |line| puts "  #{line.strip}" }
        
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include("0 examples, 0 failures, 1 error occurred outside of examples")
        expect(stdout).to include("SyntaxError")
      end
    end

    it "shows behavior for syntax errors in code under test" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/calculator_spec.rb")
        
        puts "\n=== Syntax error in code under test ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?.*error/).first&.strip}"
        puts "Error message in STDOUT: #{stdout.include?('SyntaxError')}"
        puts "STDERR: #{stderr.inspect}"
        
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include("0 examples, 0 failures, 1 error occurred outside of examples")
        expect(stdout).to include("SyntaxError")
      end
    end

    it "shows behavior for LoadError" do
      chdir(fixture_dir) do
        stdout, stderr, status = Open3.capture3("bundle", "exec", "rspec", "spec/load_error_spec.rb")
        
        puts "\n=== LoadError ==="
        puts "Exit code: #{status.exitstatus}"
        puts "Summary: #{stdout.lines.grep(/\d+ examples?.*error/).first&.strip}"
        puts "Error message in STDOUT: #{stdout.include?('LoadError')}"
        puts "STDERR: #{stderr.inspect}"
        
        expect(status.exitstatus).to eq(1)
        expect(stdout).to include("0 examples, 0 failures, 1 error occurred outside of examples")
        expect(stdout).to include("LoadError")
      end
    end
  end

  context "running through rux" do
    it "compares rux behavior for syntax errors" do
      chdir(fixture_dir) do
        result = run_rux_allowing_errors("spec/syntax_error_spec.rb")
        
        puts "\n=== Rux with syntax error ==="
        puts "Exit code: #{result.exit_status}"
        puts "Summary: #{result.out.lines.grep(/\d+ examples?/).first&.strip}"
        puts "Shows error details: #{result.out.include?('SyntaxError') || result.err.include?('SyntaxError')}"
        
        expect(result.exit_status).to eq(1)
        # Currently rux shows "0 examples, 0 failures, 1 error occurred outside of examples"
        # but doesn't show the actual syntax error details
      end
    end
  end
end