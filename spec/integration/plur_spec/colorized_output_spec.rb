require "spec_helper"

RSpec.describe "Plur colorized output" do
  let(:fixture_path) { File.join(__dir__, "..", "..", "fixtures", "rspec_colorized_output.txt") }
  let(:expected_output) { File.read(fixture_path) }

  before do
    expect(File.exist?(fixture_path)).to be(true)
  end

  context "with color enabled (default)" do
    it "outputs red F for failures matching rspec format" do
      failing_specs_path = project_fixture("failing_specs")
      chdir(failing_specs_path) do
        result = run_plur_allowing_errors("--color", "spec/mixed_results_spec.rb", env: {"FORCE_COLOR" => "1"})

        # Check that we have red F's for failures
        expect(result.out).to include("\e[31mF\e[0m")

        # Check that we have green dots for passes
        expect(result.out).to include("\e[32m.\e[0m")

        # Mixed results should have both passes and failures
        expect(result.out.scan("\e[31mF\e[0m").size).to be >= 2
        expect(result.out.scan("\e[32m.\e[0m").size).to be >= 3
      end
    end
  end

  context "with --no-color flag" do
    it "outputs plain text without ANSI codes" do
      failing_specs_path = project_fixture("failing_specs")
      chdir(failing_specs_path) do
        result = run_plur_allowing_errors("--no-color", "spec/mixed_results_spec.rb")

        # Should not contain any ANSI escape sequences
        expect(result.out).not_to match(/\e\[\d+m/)

        # Should still show F for failures and . for passes
        expect(result.out).to include("F")
        expect(result.out).to include(".")
      end
    end
  end
end
