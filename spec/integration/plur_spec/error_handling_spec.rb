require "spec_helper"
require "tmpdir"
require "open3"

RSpec.describe "Plur error handling" do
  describe "RSpec error behaviors" do
    let(:fixture_dir) { project_fixture("rspec-errors") }

    context "running RSpec directly (baseline behavior)" do
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
          puts "Error message in STDOUT: #{stdout.include?("SyntaxError")}"
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
          puts "Error message in STDOUT: #{stdout.include?("SyntaxError")}"
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
          puts "Error message in STDOUT: #{stdout.include?("LoadError")}"
          puts "STDERR: #{stderr.inspect}"

          expect(status.exitstatus).to eq(1)
          expect(stdout).to include("0 examples, 0 failures, 1 error occurred outside of examples")
          expect(stdout).to include("LoadError")
        end
      end
    end

    context "running through plur" do
      it "properly shows syntax error details" do
        chdir(fixture_dir) do
          result = run_plur_allowing_errors("spec/syntax_error_spec.rb")

          puts "\n=== Plur with syntax error ==="
          puts "Exit code: #{result.exit_status}"
          puts "Summary: #{result.out.lines.grep(/\d+ examples?/).first&.strip}"
          puts "Shows error details: #{result.out.include?("SyntaxError")}"

          expect(result.exit_status).to eq(1)
          expect(result.out).to include("0 examples, 0 failures, 1 error occurred outside of examples")
          expect(result.out).to include("SyntaxError")
          expect(result.out.downcase).to include("syntax error")

          # Should NOT save runtime data when tests fail to start
          expect(result.err).not_to include("Runtime data saved to:")
        end
      end

      it "verifies exit codes for different failure scenarios" do
        chdir(fixture_dir) do
          puts "\n=== Test 1: Normal test failure ==="
          result1 = run_plur_allowing_errors("spec/test_failure_spec.rb")
          puts "Exit code: #{result1.exit_status}"
          puts "Output: #{result1.out.lines.grep(/examples?.*failures?/).first}"

          puts "\n=== Test 2: Unhandled exception ==="
          result2 = run_plur_allowing_errors("spec/exception_spec.rb")
          puts "Exit code: #{result2.exit_status}"
          puts "Output: #{result2.out.lines.grep(/examples?.*failures?/).first}"

          puts "\n=== Test 3: Syntax error in spec ==="
          result3 = run_plur_allowing_errors("spec/syntax_error_spec.rb")
          puts "Exit code: #{result3.exit_status}"
          puts "Output: #{result3.out.lines.grep(/examples?.*failures?.*error/).first}"

          puts "\n=== Test 4: Syntax error in code under test ==="
          result4 = run_plur_allowing_errors("spec/calculator_spec.rb")
          puts "Exit code: #{result4.exit_status}"
          puts "Output: #{result4.out.lines.grep(/examples?.*failures?.*error/).first}"

          # All should exit with non-zero
          expect(result1.exit_status).to eq(1)
          expect(result2.exit_status).to eq(1)
          expect(result3.exit_status).to eq(1)
          expect(result4.exit_status).to eq(1)

          # Test failures and exceptions show examples
          expect(result1.out).to match(/2 examples, 1 failure/)
          expect(result2.out).to match(/2 examples, 1 failure/)

          # Syntax errors show 0 examples with errors
          expect(result3.out).to match(/0 examples, 0 failures, 1 error/)
          expect(result4.out).to match(/0 examples, 0 failures, 1 error/)
        end
      end
    end
  end

  describe "Edge cases" do
    context "when JSON output is empty or malformed" do
      it "provides helpful error message instead of raw JSON parsing error" do
        fixture_dir = project_fixture("empty_json")

        # Run plur and capture output
        chdir(fixture_dir) do
          env = {"PATH" => "#{fixture_dir}:$PATH"}
          result = run_plur_allowing_errors("test_spec.rb", env: env)
          expect(result.exit_status).not_to eq(0)

          # Most importantly, should NOT show raw JSON parsing error
          expect(result.err).not_to include("failed to parse JSON: unexpected end of JSON input")
          expect(result.out).not_to include("failed to parse JSON: unexpected end of JSON input")
        end
      end
    end

    context "when spec directory doesn't exist" do
      it "handles missing spec directory gracefully" do
        Dir.mktmpdir do |tmpdir|
          chdir(tmpdir) do
            result = run_plur_allowing_errors

            # Should handle gracefully and show appropriate message
            expect(result.out + result.err).to include("no spec files found").or include("ERROR")
          end
        end
      end
    end
  end
end
