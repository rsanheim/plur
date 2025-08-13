require "spec_helper"

RSpec.describe "plur glob pattern support" do
  before do
    expect(Dir.exist?(default_ruby_dir)).to be(true), "Test project not found at #{default_ruby_dir}"
  end

  context "with glob patterns" do
    it "expands simple glob patterns correctly" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/*_spec.rb")

        expect(result.err).to include("[dry-run] Found 10 spec files")
        expect(result.err).to include("spec/array_helpers_spec.rb")
        expect(result.err).to include("spec/calculator_spec.rb")
        expect(result.err).to include("spec/counter_spec.rb")
        expect(result.err).to include("spec/date_formatter_spec.rb")
        expect(result.err).to include("spec/plur_ruby_spec.rb")
        expect(result.err).not_to include("spec/models/user_spec.rb")
        expect(result.err).not_to include("spec/services/email_service_spec.rb")
      end
    end

    it "expands recursive glob patterns with **" do
      chdir(default_ruby_dir) do
        # Note: Without quotes, this relies on shell expansion
        # The shell must have globstar enabled (bash: shopt -s globstar, zsh: default on)
        # For consistent testing, we'll use bash with globstar
        cmd = TTY::Command.new(uuid: false, printer: :null)
        result = cmd.run!("bash", "-c", "shopt -s globstar; plur --dry-run spec/**/*_spec.rb")

        expect(result.err).to include("[dry-run] Found 12 spec files")
        expect(result.err).to include("spec/array_helpers_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
        expect(result.err).to include("spec/services/email_service_spec.rb")
      end
    end

    it "handles quoted recursive glob patterns with ** (preventing shell expansion)" do
      chdir(default_ruby_dir) do
        # Using single quotes prevents shell expansion, so plur handles the glob
        result = run_plur("--dry-run", "spec/**/*_spec.rb")

        expect(result.err).to include("[dry-run] Found 12 spec files")
        expect(result.err).to include("spec/array_helpers_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
        expect(result.err).to include("spec/services/email_service_spec.rb")
      end
    end

    it "expands multiple glob patterns" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/models/*_spec.rb", "spec/services/*_spec.rb")

        expect(result.err).to include("[dry-run] Found 2 spec files")
        expect(result.err).to include("spec/models/user_spec.rb")
        expect(result.err).to include("spec/services/email_service_spec.rb")
      end
    end

    it "handles character class patterns with []" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/[cs]*_spec.rb")

        expect(result.err).to include("[dry-run] Found 3 spec files")
        expect(result.err).to include("spec/calculator_spec.rb")
        expect(result.err).to include("spec/counter_spec.rb")
        expect(result.err).to include("spec/string_utils_spec.rb")
      end
    end

    it "handles specific file paths without globs" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "spec/calculator_spec.rb", "spec/counter_spec.rb")

        expect(result.err).to include("[dry-run] Found 2 spec files")
        expect(result.err).to include("spec/calculator_spec.rb")
        expect(result.err).to include("spec/counter_spec.rb")
      end
    end

    it "warns about non-spec files but still passes them to RSpec" do
      chdir(default_ruby_dir) do
        # Create a temporary non-spec file (doesn't end with _spec.rb)
        File.write("spec/helper.rb", "# helper file")

        begin
          result = run_plur("--dry-run", "spec/helper.rb")

          # Should succeed with dry-run (matching RSpec's behavior of accepting any file)
          expect(result.success?).to be true
          expect(result.err).to include("Warning: spec/helper.rb does not end with _spec.rb")
          # Should still include the file in dry-run output
          expect(result.err).to include("[dry-run] Found 1 spec files")
          expect(result.err).to include("spec/helper.rb")
        ensure
          File.delete("spec/helper.rb") if File.exist?("spec/helper.rb")
        end
      end
    end

    it "returns error for non-existent files" do
      chdir(default_ruby_dir) do
        result = run_plur_allowing_errors("--dry-run", "spec/nonexistent_spec.rb")

        expect(result.success?).to be false
        expect(result.out + result.err).to include("file not found: spec/nonexistent_spec.rb")
      end
    end

    it "returns error when no files match glob pattern" do
      chdir(default_ruby_dir) do
        result = run_plur_allowing_errors("--dry-run", "spec/xyz*_spec.rb")

        expect(result.success?).to be false
        expect(result.out + result.err).to include("no test files found matching provided patterns")
      end
    end
  end

  context "without arguments (auto-discovery)" do
    it "finds all spec files recursively" do
      chdir(default_ruby_dir) do
        result = run_plur("--dry-run")

        expect(result.err).to include("[dry-run] Found 12 spec files")
        expect(result.err).to include("spec/array_helpers_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
        expect(result.err).to include("spec/services/email_service_spec.rb")
        expect(result.err).to include("spec/example_scenarios_spec.rb")
      end
    end
  end
end
