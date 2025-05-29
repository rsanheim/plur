require "spec_helper"
require "tmpdir"

RSpec.describe "Rux error handling" do
  context "when JSON output is empty or malformed" do
    it "provides helpful error message instead of raw JSON parsing error" do
      fixture_dir = File.join(__dir__, "..", "test_fixtures", "empty_json")

      # Run rux and capture output
      chdir(fixture_dir) do
        env = {"PATH" => "#{fixture_dir}:$PATH"}
        stdout, stderr, status = Open3.capture3(env, "#{rux_binary} test_spec.rb")
        expect(status.exitstatus).not_to eq(0)

        # TODO we gotta implement error counts
        # expect(stdout).to include("1 error")

        # Most importantly, should NOT show raw JSON parsing error
        expect(stderr).not_to include("failed to parse JSON: unexpected end of JSON input")
        expect(stdout).not_to include("failed to parse JSON: unexpected end of JSON input")
      end
    end
  end

  context "when spec directory doesn't exist" do
    it "handles missing spec directory gracefully" do
      Dir.mktmpdir do |tmpdir|
        Dir.chdir(tmpdir) do
          output = `#{rux_binary} 2>&1`

          # Should handle gracefully and show appropriate message
          expect(output).to include("no spec files found").or include("ERROR")
        end
      end
    end
  end
end
