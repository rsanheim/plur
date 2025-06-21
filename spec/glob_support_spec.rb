require "spec_helper"
require "open3"

RSpec.describe "rux glob pattern support" do
  before do
    expect(File.exist?(rux_binary)).to be(true), "Rux binary not found at #{rux_binary}"
    expect(Dir.exist?(default_ruby_dir)).to be(true), "Test project not found at #{default_ruby_dir}"
  end

  context "with glob patterns" do
    it "expands simple glob patterns correctly" do
      Dir.chdir(default_ruby_dir) do
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run spec/*_spec.rb")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 10 spec files")
        expect(stderr).to include("spec/array_helpers_spec.rb")
        expect(stderr).to include("spec/calculator_spec.rb")
        expect(stderr).to include("spec/counter_spec.rb")
        expect(stderr).to include("spec/date_formatter_spec.rb")
        expect(stderr).to include("spec/rux_ruby_spec.rb")
        expect(stderr).not_to include("spec/models/user_spec.rb")
        expect(stderr).not_to include("spec/services/email_service_spec.rb")
      end
    end

    it "expands recursive glob patterns with **" do
      Dir.chdir(default_ruby_dir) do
        # Note: Without quotes, this relies on shell expansion
        # The shell must have globstar enabled (bash: shopt -s globstar, zsh: default on)
        # For consistent testing, we'll use bash with globstar
        _, stderr, status = Open3.capture3("bash", "-c", "shopt -s globstar; #{rux_binary} --dry-run spec/**/*_spec.rb")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 12 spec files")
        expect(stderr).to include("spec/array_helpers_spec.rb")
        expect(stderr).to include("spec/models/user_spec.rb")
        expect(stderr).to include("spec/services/email_service_spec.rb")
      end
    end

    it "handles quoted recursive glob patterns with ** (preventing shell expansion)" do
      Dir.chdir(default_ruby_dir) do
        # Using single quotes prevents shell expansion, so rux handles the glob
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run 'spec/**/*_spec.rb'")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 12 spec files")
        expect(stderr).to include("spec/array_helpers_spec.rb")
        expect(stderr).to include("spec/models/user_spec.rb")
        expect(stderr).to include("spec/services/email_service_spec.rb")
      end
    end

    it "expands multiple glob patterns" do
      Dir.chdir(default_ruby_dir) do
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run spec/models/*_spec.rb spec/services/*_spec.rb")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 2 spec files")
        expect(stderr).to include("spec/models/user_spec.rb")
        expect(stderr).to include("spec/services/email_service_spec.rb")
      end
    end

    it "handles character class patterns with []" do
      Dir.chdir(default_ruby_dir) do
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run spec/[cs]*_spec.rb")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 3 spec files")
        expect(stderr).to include("spec/calculator_spec.rb")
        expect(stderr).to include("spec/counter_spec.rb")
        expect(stderr).to include("spec/string_utils_spec.rb")
      end
    end

    it "handles specific file paths without globs" do
      Dir.chdir(default_ruby_dir) do
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run spec/calculator_spec.rb spec/counter_spec.rb")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 2 spec files")
        expect(stderr).to include("spec/calculator_spec.rb")
        expect(stderr).to include("spec/counter_spec.rb")
      end
    end

    it "warns about non-spec files and returns error when no specs found" do
      Dir.chdir(default_ruby_dir) do
        # Create a temporary non-spec file (doesn't end with _spec.rb)
        File.write("spec/helper.rb", "# helper file")

        begin
          output, stderr, status = Open3.capture3("#{rux_binary} --dry-run spec/helper.rb")

          expect(status).not_to be_success  # Fails when no spec files found
          expect(stderr).to include("Warning: spec/helper.rb does not end with _spec.rb")
          # Kong logs errors through the logger with a specific format
          expect(output + stderr).to include("no spec files found matching provided patterns")
        ensure
          File.delete("spec/helper.rb") if File.exist?("spec/helper.rb")
        end
      end
    end

    it "returns error for non-existent files" do
      Dir.chdir(default_ruby_dir) do
        output, _, status = Open3.capture3("#{rux_binary} --dry-run spec/nonexistent_spec.rb 2>&1")

        expect(status).not_to be_success
        expect(output).to include("file not found: spec/nonexistent_spec.rb")
      end
    end

    it "returns error when no files match glob pattern" do
      Dir.chdir(default_ruby_dir) do
        output, _, status = Open3.capture3("#{rux_binary} --dry-run spec/xyz*_spec.rb 2>&1")

        expect(status).not_to be_success
        expect(output).to include("no spec files found matching provided patterns")
      end
    end
  end

  context "without arguments (auto-discovery)" do
    it "finds all spec files recursively" do
      Dir.chdir(default_ruby_dir) do
        _, stderr, status = Open3.capture3("#{rux_binary} --dry-run")

        expect(status).to be_success
        expect(stderr).to include("[dry-run] Found 12 spec files")
        expect(stderr).to include("spec/array_helpers_spec.rb")
        expect(stderr).to include("spec/models/user_spec.rb")
        expect(stderr).to include("spec/services/email_service_spec.rb")
        expect(stderr).to include("spec/example_scenarios_spec.rb")
      end
    end
  end
end
