require_relative "../../spec_helper"

RSpec.describe "Minitest Integration" do
  before(:all) do
    chdir(project_fixture!("minitest-success")) do
      Bundler.with_unbundled_env do
        system("bundle install", exception: true)
      end
    end
  end

  context "with minitest-success project" do
    let(:project_dir) { project_fixture!("minitest-success") }

    it "runs minitest tests successfully" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--type", "minitest", "-n", "1")
          expect(result).to be_success
          expect(result.out).to include("plur version")
          expect(result.out).to include("Running 2 spec files")
          # Minitest shows the final summary
          expect(result.out).to match(/\d+ runs?, \d+ assertions?, 0 failures?, 0 errors?/)
        end
      end
    end

    it "formats minitest duration using RSpec-style formatting" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--type", "minitest", "-n", "1")
          expect(result).to be_success
          # Should use "seconds" not "s" suffix
          expect(result.out).to match(/Finished in \d+(?:\.\d{1,5})? seconds/)
          # Should NOT use the old "Xs" format
          expect(result.out).not_to match(/Finished in [\d.]+s\./)
        end
      end
    end

    it "discovers minitest test files" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--type", "minitest", "--dry-run")
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
          result = run_plur("--type", "minitest", "-n", "1", allow_error: true)
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
end
