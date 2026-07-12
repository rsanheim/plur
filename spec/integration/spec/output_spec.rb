require "spec_helper"

RSpec.describe "plur spec output handling" do
  describe "concurrent output handling" do
    it "produces valid output with high worker count" do
      Dir.chdir(default_ruby_dir) do
        # Run with many workers to stress-test the output handling
        result = run_plur("-n", "8")

        # Count the dots in output
        dot_count = result.out.scan(".").count

        # Should have at least some dots (examples)
        expect(dot_count).to be > 0

        # Output should still be valid
        expect(result.out).to include("examples")
        expect(result.out).to include("failures")
      end
    end

    it "maintains colored output when supported" do
      Dir.chdir(default_ruby_dir) do
        # Force color output
        result = run_plur("-n", "4", env: {"FORCE_COLOR" => "1"})

        # Should contain ANSI color codes for green dots
        expect(result.out).to include("\e[32m.\e[0m")
      end
    end

    it "handles mixed output types correctly" do
      # Create a temporary spec file with actual failures
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "mixed_spec.rb"), <<~RUBY)
          RSpec.describe 'Mixed' do
            it 'passes' do
              expect(1).to eq(1)
            end
            
            it 'fails' do
              expect(1).to eq(2)
            end
            
            it 'also passes' do
              expect(true).to be true
            end
            
            it 'also fails' do
              expect(false).to be true
            end
          end
        RUBY

        Dir.chdir(tmpdir) do
          # Run specs that include failures
          result = run_plur("-n", "2", "mixed_spec.rb", allow_error: true)

          # Should show both dots and F's
          expect(result.out).to match(/[.F]+/)

          # Should exit with status 1 for failures
          expect(result.status).to eq(1)
        end
      end
    end
  end

  describe "PLUR_DEBUG environment variable" do
    it "enables debug logging when set to 1" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", env: {"PLUR_DEBUG" => "1"})

        expect(result.err).to include("DEBUG")
        expect(result.err).to include("discovered test files")
      end
    end

    it "stays quiet when unset" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run")

        expect(result.err).not_to include("DEBUG")
      end
    end

    it "stays quiet when set to 0" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", env: {"PLUR_DEBUG" => "0"})

        expect(result.err).not_to include("DEBUG")
      end
    end
  end

  describe "worker command errors" do
    it "keeps worker stderr off stdout when the command exits before test events" do
      tmp_root = ROOT_PATH.join("tmp")
      FileUtils.mkdir_p(tmp_root)

      Dir.mktmpdir("worker-error-output-", tmp_root.to_s) do |tmpdir|
        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "boom_spec.rb"), "# target placeholder\n")
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          use = "boom"
          color = "never"

          [job.boom]
          framework = "rspec"
          cmd = ["sh", "-c", "echo WORKER_STDERR_MARKER >&2; exit 42", "boom"]
          target_pattern = "spec/**/*_spec.rb"
        TOML

        stdout, stderr, status = Dir.chdir(tmpdir) do
          Open3.capture3(plur_binary, "-n", "1")
        end

        expect(status.exitstatus).to eq(1)
        expect(stderr).to include("WORKER_STDERR_MARKER")
        expect(stdout).not_to include("WORKER_STDERR_MARKER")
        expect(stdout).to include("0 examples, 0 failures")
      end
    end

    it "prints worker startup errors to stderr" do
      tmp_root = ROOT_PATH.join("tmp")
      FileUtils.mkdir_p(tmp_root)

      Dir.mktmpdir("worker-startup-error-", tmp_root.to_s) do |tmpdir|
        FileUtils.mkdir_p(File.join(tmpdir, "spec"))
        File.write(File.join(tmpdir, "spec", "missing_spec.rb"), "# target placeholder\n")
        File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
          use = "missing"
          color = "never"

          [job.missing]
          framework = "rspec"
          cmd = ["./definitely-missing-command"]
          target_pattern = "spec/**/*_spec.rb"
        TOML

        stdout, stderr, status = Dir.chdir(tmpdir) do
          Open3.capture3(plur_binary, "-n", "1")
        end

        expect(status.exitstatus).to eq(1)
        expect(stderr).to include("Error: fork/exec ./definitely-missing-command")
        expect(stdout).not_to include("fork/exec ./definitely-missing-command")
        expect(stdout).to include("0 examples, 0 failures")
      end
    end
  end
end
