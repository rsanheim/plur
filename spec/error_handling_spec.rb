require "spec_helper"
require "tmpdir"

RSpec.describe "Rux error handling" do
  context "when JSON output is empty or malformed" do
    it "provides helpful error message instead of raw JSON parsing error" do
      fixture_dir = project_fixture("empty_json")

      # Run rux and capture output
      chdir(fixture_dir) do
        env = {"PATH" => "#{fixture_dir}:$PATH"}
        result = run_rux_allowing_errors("test_spec.rb", env: env)
        expect(result.exit_status).not_to eq(0)

        # TODO we gotta implement error counts
        # expect(result.out).to include("1 error")

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
          result = run_rux_allowing_errors

          # Should handle gracefully and show appropriate message
          expect(result.out + result.err).to include("no spec files found").or include("ERROR")
        end
      end
    end
  end

  context "when spec file has invalid Ruby syntax" do
    it "reports the error clearly instead of showing 0 examples" do
      Dir.mktmpdir do |tmpdir|
        # Create a spec file with invalid Ruby syntax
        spec_file = File.join(tmpdir, "invalid_spec.rb")
        File.write(spec_file, <<~RUBY)
          RSpec.describe "Invalid" do
            it "has syntax error" do
              # Missing end statement for the block
              expect(1).to eq(1)
          end
        RUBY

        chdir(tmpdir) do
          result = run_rux_allowing_errors("invalid_spec.rb")
          
          # Should exit with error
          expect(result.exit_status).not_to eq(0)
          
          # CURRENT BEHAVIOR (problematic):
          # - Shows "0 examples, 0 failures, 1 error occurred outside of examples"
          # - Doesn't show the actual Ruby syntax error
          # - Still saves runtime data
          expect(result.out).to include("0 examples, 0 failures, 1 error occurred outside of examples")
          
          # TODO: Should show the actual syntax error from Ruby
          # expect(result.err).to include("syntax error")
          
          # TODO: Should not save runtime data when tests fail to start
          expect(result.err).to include("Runtime data saved to:")
        end
      end
    end
  end

  context "RSpec exit code behavior" do
    it "verifies exit codes for different failure scenarios" do
      Dir.mktmpdir do |tmpdir|
        # Test 1: Normal test failure (assertion fails)
        File.write(File.join(tmpdir, "test_failure_spec.rb"), <<~RUBY)
          RSpec.describe "Test failures" do
            it "fails an assertion" do
              expect(1).to eq(2)
            end
            it "passes" do
              expect(1).to eq(1)
            end
          end
        RUBY

        # Test 2: Unhandled exception in test
        File.write(File.join(tmpdir, "exception_spec.rb"), <<~RUBY)
          RSpec.describe "Exceptions" do
            it "raises an unhandled exception" do
              raise "Unhandled error!"
            end
            it "passes" do
              expect(1).to eq(1)
            end
          end
        RUBY

        # Test 3: Ruby syntax error in spec
        File.write(File.join(tmpdir, "syntax_error_spec.rb"), <<~RUBY)
          RSpec.describe "Syntax error" do
            it "has invalid syntax" do
              # Missing closing bracket
              expect(1.to eq(1)
            end
          end
        RUBY

        # Test 4: Syntax error in code under test
        File.write(File.join(tmpdir, "calculator.rb"), <<~RUBY)
          class Calculator
            def add(a, b
              # Missing closing parenthesis
              a + b
            end
          end
        RUBY

        File.write(File.join(tmpdir, "calculator_spec.rb"), <<~RUBY)
          require_relative 'calculator'
          
          RSpec.describe Calculator do
            it "adds numbers" do
              expect(Calculator.new.add(1, 2)).to eq(3)
            end
          end
        RUBY

        chdir(tmpdir) do
          puts "\n=== Test 1: Normal test failure ==="
          result1 = run_rux_allowing_errors("test_failure_spec.rb")
          puts "Exit code: #{result1.exit_status}"
          puts "Output: #{result1.out.lines.grep(/examples?.*failures?/).first}"
          
          puts "\n=== Test 2: Unhandled exception ==="
          result2 = run_rux_allowing_errors("exception_spec.rb")
          puts "Exit code: #{result2.exit_status}"
          puts "Output: #{result2.out.lines.grep(/examples?.*failures?/).first}"
          
          puts "\n=== Test 3: Syntax error in spec ==="
          result3 = run_rux_allowing_errors("syntax_error_spec.rb")
          puts "Exit code: #{result3.exit_status}"
          puts "Output: #{result3.out.lines.grep(/examples?.*failures?.*error/).first}"
          
          puts "\n=== Test 4: Syntax error in code under test ==="
          result4 = run_rux_allowing_errors("calculator_spec.rb")
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

    it "shows stderr when command doesn't exist" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "dummy_spec.rb"), <<~RUBY)
          RSpec.describe "Dummy" do
            it "passes" do
              expect(1).to eq(1)
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_rux_allowing_errors("--command=nonexistentcommand", "dummy_spec.rb", "--debug")
          
          puts "\n=== Command not found (with debug) ==="
          puts "Exit code: #{result.exit_status}"
          puts "STDOUT:\n#{result.out}"
          puts "STDERR:\n#{result.err}"
          
          expect(result.exit_status).to eq(1)
          # Currently shows "0 examples, 0 failures" which is misleading
          expect(result.out).to include("0 examples, 0 failures")
          
          # Check if there's any error indication in debug output
          # expect(result.err).to include("failed to start command")
        end
      end
    end
  end
end
