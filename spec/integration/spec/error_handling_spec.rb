require "spec_helper"
require "tmpdir"
require "open3"

RSpec.describe "Plur error handling" do
  let(:fixture_dir) { project_fixture("rspec-errors") }

  it "properly shows syntax error details" do
    chdir(fixture_dir) do
      result = run_plur_allowing_errors("spec/syntax_error_spec.rb")

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
      result1 = run_plur_allowing_errors("spec/test_failure_spec.rb")
      result2 = run_plur_allowing_errors("spec/exception_spec.rb")
      result3 = run_plur_allowing_errors("spec/syntax_error_spec.rb")
      result4 = run_plur_allowing_errors("spec/calculator_spec.rb")

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

  it "includes errors outside of examples in multi-worker summaries" do
    chdir(fixture_dir) do
      result = run_plur_allowing_errors("-n", "2", "--color=never",
        "spec/passing_spec.rb", "spec/load_error_spec.rb")

      expect(result.exit_status).to eq(1)
      expect(result.out).to match(/2 examples, 0 failures, 1 error occurred outside of examples/)
    end
  end

  it "shows an errored worker's pre-crash stdout once, alongside the load error" do
    chdir(fixture_dir) do
      error_file = "spec/stdout_then_load_error_spec.rb"
      File.write(error_file, <<~RUBY)
        puts "STDOUT_BEFORE_CRASH"
        require "this_gem_does_not_exist_surely"
      RUBY

      result = run_plur_allowing_errors("-n", "2", "--color=never",
        "spec/passing_spec.rb", error_file)

      expect(result.exit_status).to eq(1)
      # The load error itself must still be reported...
      expect(result.out).to include("cannot load such file")
      # ...but the stdout printed before the crash streamed live and must
      # not be re-printed with the error summary.
      expect(result.out.scan("STDOUT_BEFORE_CRASH").length).to eq(1),
        "errored worker stdout should appear once (streamed live), not re-dumped"
    ensure
      FileUtils.rm_f(error_file)
    end
  end

  context "when JSON output is empty or malformed" do
    it "provides helpful error message instead of raw JSON parsing error" do
      fixture_dir = project_fixture("empty_json")

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
          expect(result.err).to include("Error: no default spec/test files found")
          expect(result.err).not_to include("ERROR - Command failed")
        end
      end
    end
  end

  it "prints missing target errors as plain user-facing errors" do
    result = run_plur_allowing_errors("--dry-run", "spec/nonexistent_spec.rb")

    expect(result.exit_status).to eq(1)
    expect(result.err).to include("Error: file not found: spec/nonexistent_spec.rb")
    expect(result.err).not_to include("ERROR - Command failed")
    expect(result.err).not_to match(/^\d{2}:\d{2}:\d{2} - ERROR/)
  end
end
