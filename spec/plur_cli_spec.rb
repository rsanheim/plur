require "spec_helper"

RSpec.describe "Plur CLI behavior" do
  let(:calculator_spec_examples) { 5 }

  describe "dry-run functionality" do
    it "runs dry-run with no arguments" do
      result = run_plur("--dry-run")

      expect(result.err).to match(%r{\[dry-run\] Found \d+ spec files, running in parallel})
      expect(result.err).to include("rspec")
    end

    it "shows overridden command if defined" do
      result = run_plur("spec", "--dry-run", "--command=bin/rspec")

      expect(result.err).to include("[dry-run]")
      expect(result.err).to include("bin/rspec")
      expect(result.err).to_not include("bundle exec rspec")
    end

    it "runs dry-run with specific spec file" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to include("calculator_spec.rb")
      end
    end

    it "runs dry-run with multiple spec files" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/calculator_spec.rb", "spec/rux_ruby_spec.rb")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to include("calculator_spec.rb")
        expect(result.err).to include("rux_ruby_spec.rb")
      end
    end

    it "runs dry-run with --auto flag" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "--auto", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run]")
      end
    end

    it "runs dry-run with auto-discovery in rux-ruby" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to match(/Found \d+ spec files/)
      end
    end
  end

  describe "actual test execution" do
    it "runs all specs in parallel with auto-discovery" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur

        expect(result.out).to match(/Running \d+ spec files in parallel/)
        expect(result.out).to include("66 examples, 0 failures")
      end
    end

    it "runs with --auto flag (bundle install + run)" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--auto")
        expect(result.out).to include("Bundle complete!")
        expect(result.out).to include("examples, 0 failures")
      end
    end

    it "runs specific spec file" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("spec/calculator_spec.rb")

        expect(result.out).to include("#{calculator_spec_examples} examples, 0 failures")
      end
    end

    it "uses debug mode for -d" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("spec", "-d", "spec/calculator_spec.rb")

        expect(result.err).to include("bundle exec rspec")
        expect(result.out).to include("#{calculator_spec_examples} examples, 0 failures")
      end
    end

    it "uses debug mode for --debug" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("spec", "--debug", "spec/calculator_spec.rb")

        expect(result.err).to include("bundle exec rspec")
        expect(result.out).to include("#{calculator_spec_examples} examples, 0 failures")
      end
    end

    it "uses the configured command if defined" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("spec", "--debug", "--command=rspec")

        expect(result.err).to include("rspec -r")
        expect(result.err).not_to include("bundle exec")
        expect(result.out).to include("examples, 0 failures")
      end
    end

    # TODO - we need better handling and feedback here if a command does not exist
    it "errors if configured command is not found" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur_allowing_errors("spec", "--debug", "--command=rspecx")
        expect(result).to_not be_success
        expect(result.exit_status).to be_nonzero
        # Error messages are displayed after the summary
        expect(result.out).to include("Error: failed to start command")
        expect(result.out).to include("rspecx")
        expect(result.out).to include("not found")
      end
    end

    it "provides interleaved output from parallel execution" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("-n", "2")

        expect(result.out).to match(/\.+/)
        expect(result.out).to include("examples, 0 failures")
      end
    end
  end
end
