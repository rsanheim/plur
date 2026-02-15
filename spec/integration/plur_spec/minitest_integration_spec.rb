require_relative "../../spec_helper"

RSpec.describe "Minitest Integration" do
  before(:all) do
    chdir(project_fixture!("minitest-success")) do
      Bundler.with_unbundled_env do
        system("bundle check", out: File::NULL, err: File::NULL) ||
          system("bundle install", out: File::NULL, err: File::NULL, exception: true)
      end
    end
  end

  context "with minitest-success project" do
    let(:project_dir) { project_fixture!("minitest-success") }

    it "runs minitest tests successfully" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1")
          expect(result).to be_success
          expect(result.err).to include("plur version")
          expect(result.err).to include("Running 2 tests [minitest]")
          # Minitest shows the final summary
          expect(result.out).to match(/\d+ runs?, \d+ assertions?, 0 failures?, 0 errors?/)
        end
      end
    end

    it "formats minitest duration using RSpec-style formatting" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1")
          expect(result).to be_success
          # Should use "seconds" not "s" suffix
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          # Should NOT use the old "Xs" format
          expect(result.out).not_to match(/Finished in [\d.]+s\./)
        end
      end
    end

    it "displays correct progress dot count with puts interleaving" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", "--no-color")
          expect(result).to be_success
          # All 8 progress dots should appear on the first line,
          # even though some are on lines interleaved with puts output
          progress_line = result.out.split("\n").first
          expect(progress_line).to eq("." * 8)
        end
      end
    end

    it "discovers minitest test files" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "--dry-run", "-n", "2")
          expect(result).to be_success
          expect(result.err).to include("test/calculator_test.rb")
          expect(result.err).to include("test/string_helper_test.rb")
        end
      end
    end
  end

  context "with minitest-failures project" do
    let(:project_dir) { project_fixture!("minitest-failures") }

    it "reports minitest test failures" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "1", allow_error: true)
          expect(result).to be_failure
          # Check for failure output in stdout (not stderr)
          # In parallel execution, individual failure details are not shown
          # Check that the summary shows failures
          expect(result.out).to match(/\d+ failures?/)
        end
      end
    end
  end

  context "with auto-detection" do
    it "detects minitest project by test directory" do
      chdir(project_fixture!("minitest-success")) do
        Bundler.with_unbundled_env do
          result = run_plur("--dry-run")
          expect(result).to be_success
          # Should use minitest commands
          expect(result.err).to include("ruby -Itest")
        end
      end
    end
  end

  context "with parallel workers and stdout interleaving" do
    let(:project_dir) { project_fixture!("minitest-success") }

    it "displays correct progress and summary with multiple workers" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--use", "minitest", "-n", "2", "--no-color")
          expect(result).to be_success

          # With 2 workers (1 file each), both producing mixed progress lines,
          # all 8 progress dots should appear on the first stdout line
          progress_line = result.out.split("\n").first
          expect(progress_line).to eq("." * 8)

          # Summary should aggregate correctly across workers
          expect(result.out).to include("8 runs")
          expect(result.out).to include("23 assertions")
          expect(result.out).to include("0 failures")
          expect(result.out).to include("0 errors")
        end
      end
    end
  end

  context "with grouped minitest command output" do
    let(:project_dir) { project_fixture!("minitest-success") }
    let(:minitest_seed) { "8378" }

    def normalize_minitest_output(output)
      output
        .gsub(/Run options: --seed \d+/, "Run options: --seed [SEED]")
        .gsub(/Finished in [\d.]+s, [\d.]+ runs\/s, [\d.]+ assertions\/s\./,
          "Finished in [DURATION]s, [RUNS_PER_SEC] runs/s, [ASSERTIONS_PER_SEC] assertions/s.")
    end

    def normalize_grouped_minitest_snapshot(snapshot)
      snapshot.merge(
        "args" => ["[MINITEST_COMPARISON_COMMAND]"],
        "stdout" => normalize_minitest_output(snapshot.fetch("stdout", "")),
        "stderr" => ""
      )
    end

    def grouped_minitest_command(seed:)
      # Use a single ruby command with a quoted -e script to require each file.
      [
        "bundle", "exec", "ruby",
        "-Itest",
        "-e", '["calculator_test", "string_helper_test"].each { |f| require f }',
        "--", "--seed", seed
      ]
    end

    def plur_minitest_serial_command
      [plur_binary, "--use", "minitest", "-n", "1", "--no-color"]
    end

    it "records grouped minitest output from a ruby -e require list" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          Backspin.run(
            grouped_minitest_command(seed: minitest_seed),
            name: "minitest_grouped_ruby_command_output",
            filter: ->(snapshot) { normalize_grouped_minitest_snapshot(snapshot) }
          )
        end
      end
    end

    it "compares plur -n1 output to grouped minitest output" do
      pending("plur output does not match raw minitest output yet")

      chdir(project_dir) do
        Bundler.with_unbundled_env do
          Backspin.run(
            plur_minitest_serial_command,
            name: "minitest_grouped_ruby_command_output",
            filter: ->(snapshot) { normalize_grouped_minitest_snapshot(snapshot) }
          )
        end
      end
    end
  end
end
