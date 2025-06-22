require "spec_helper"
require "open3"

RSpec.describe "Rux CLI behavior" do
  CALCULATOR_SPEC_EXAMPLES = 5

  describe "dry-run functionality" do
    it "runs dry-run with no arguments" do
      result = run_rux("--dry-run")

      expect(result.err).to match(%r|\[dry-run\] Found \d+ spec files, running in parallel|)
      expect(result.err).to include("rspec")
    end

    it "shows overridden command if defined" do
      result = run_rux("spec", "--dry-run", "--command=bin/rspec")

      expect(result.err).to include("[dry-run]")
      expect(result.err).to include("bin/rspec")
      expect(result.err).to_not include("bundle exec rspec")
    end

    it "runs dry-run with specific spec file" do
      chdir(default_ruby_dir) do
        result = run_rux("--dry-run", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to include("calculator_spec.rb")
      end
    end

    it "runs dry-run with multiple spec files" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux( "--dry-run", "spec/calculator_spec.rb", "spec/rux_ruby_spec.rb")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to include("calculator_spec.rb")
        expect(result.err).to include("rux_ruby_spec.rb")
      end
    end

    it "runs dry-run with --auto flag" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("--dry-run", "--auto", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run]")
      end
    end

    it "runs dry-run with auto-discovery in rux-ruby" do
      chdir(default_ruby_dir) do
        result = run_rux("--dry-run")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to match(/Found \d+ spec files/)
      end
    end
  end

  describe "actual test execution" do
    it "runs all specs in parallel with auto-discovery" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux

        expect(result.out).to match(/Running \d+ spec files in parallel/)
        expect(result.out).to include("66 examples, 0 failures")
      end
    end

    it "runs with --auto flag (bundle install + run)" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("--auto")
        expect(result.out).to include("Bundle complete!")
        expect(result.out).to include("examples, 0 failures")
      end
    end

    it "runs specific spec file" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("spec/calculator_spec.rb")

        expect(result.out).to include("#{CALCULATOR_SPEC_EXAMPLES} examples, 0 failures")
      end
    end

    it "shows the actual command(s) run when --debug is enabled" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("spec", "--debug", "spec/calculator_spec.rb")

        expect(result.err).to include("bundle exec rspec")
        expect(result.out).to include("#{CALCULATOR_SPEC_EXAMPLES} examples, 0 failures")
      end
    end

    it "uses the configured command if defined" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("spec", "--debug", "--command=rspec")

        expect(result.err).to include("rspec -r")
        expect(result.err).not_to include("bundle exec")
        expect(result.out).to include("examples, 0 failures")
      end
    end

    # TODO - we need better handling and feedback here if a command does not exist
    it "errors if configured command is not found" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux_allowing_errors("spec", "--debug", "--command=rspecx")
        expect(result).to_not be_success
        expect(result.exit_status).to be_nonzero
        expect(result.err).to include("rspecx")
        expect(result.err).to include("not found")
      end
    end

    it "provides interleaved output from parallel execution" do
      Dir.chdir(default_ruby_dir) do
        result = run_rux("-n", "2")

        expect(result.out).to match(/\.+/)
        expect(result.out).to include("examples, 0 failures")
      end
    end
  end
end
