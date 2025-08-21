require "spec_helper"

RSpec.describe "Plur CLI behavior" do
  let(:calculator_spec_examples) { 5 }

  describe "dry-run functionality" do
    it "runs dry-run with no arguments" do
      result = run_plur("--dry-run")

      expect(result.err).to match(%r{\[dry-run\] Running \d+ specs in parallel})
      expect(result.err).to include("rspec")
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
        result = run_plur("--dry-run", "spec/calculator_spec.rb", "spec/plur_ruby_spec.rb")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to include("calculator_spec.rb")
        expect(result.err).to include("plur_ruby_spec.rb")
      end
    end

    it "runs dry-run with --auto flag" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "--auto", "spec/calculator_spec.rb")

        expect(result.err).to include("[dry-run]")
      end
    end

    it "runs dry-run with auto-discovery in plur-ruby" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run")

        expect(result.err).to include("[dry-run]")
        expect(result.err).to match(/Running \d+ specs in parallel/)
      end
    end
  end

  describe "actual test execution" do
    it "runs all specs in parallel with auto-discovery" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur

        expect(result.err).to match(/Running \d+ specs in parallel/)
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

        expect(result.err).to include("DEBUG")
        expect(result.err).to include("rspec")
        expect(result.out).to include("#{calculator_spec_examples} examples, 0 failures")
      end
    end
  end
end
