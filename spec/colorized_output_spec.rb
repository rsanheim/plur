require "spec_helper"

RSpec.describe "Rux colorized output" do
  let(:fixture_path) { File.join(__dir__, "fixtures", "rspec_colorized_output.txt") }
  let(:expected_output) { File.read(fixture_path) }
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }
  let(:test_project_path) { File.join(__dir__, "..", "rux-ruby") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
    expect(File.exist?(fixture_path)).to be(true)
  end

  context "with color enabled (default)" do
    it "outputs red F for failures matching rspec format" do
      failing_specs_path = File.join(__dir__, "..", "test_fixtures", "failing_specs")
      Dir.chdir(failing_specs_path) do
        output = `FORCE_COLOR=1 #{rux_binary} --color spec/mixed_results_spec.rb 2>&1`

        # Check that we have red F's for failures
        expect(output).to include("\e[31mF\e[0m")

        # Check that we have green dots for passes
        expect(output).to include("\e[32m.\e[0m")

        # Mixed results should have both passes and failures
        expect(output.scan("\e[31mF\e[0m").size).to be >= 2
        expect(output.scan("\e[32m.\e[0m").size).to be >= 3
      end
    end
  end

  context "with --no-color flag" do
    it "outputs plain text without ANSI codes" do
      failing_specs_path = File.join(__dir__, "..", "test_fixtures", "failing_specs")
      Dir.chdir(failing_specs_path) do
        output = `#{rux_binary} --no-color spec/mixed_results_spec.rb 2>&1`

        # Should not contain any ANSI escape sequences
        expect(output).not_to match(/\e\[\d+m/)

        # Should still show F for failures and . for passes
        expect(output).to include("F")
        expect(output).to include(".")
      end
    end
  end

  context "with --no-colour flag (British spelling)" do
    it "outputs plain text without ANSI codes" do
      failing_specs_path = File.join(__dir__, "..", "test_fixtures", "failing_specs")
      Dir.chdir(failing_specs_path) do
        output = `#{rux_binary} --no-colour spec/mixed_results_spec.rb 2>&1`

        # Should not contain any ANSI escape sequences
        expect(output).not_to match(/\e\[\d+m/)

        # Should still show F for failures and . for passes
        expect(output).to include("F")
        expect(output).to include(".")
      end
    end
  end
end
