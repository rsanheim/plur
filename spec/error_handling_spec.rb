require "spec_helper"
require "tmpdir"

RSpec.describe "Rux error handling" do
  context "when JSON output is empty or malformed" do
    it "provides helpful error message instead of raw JSON parsing error" do
      fixture_dir = project_fixture("empty_json")

      # Run rux and capture output
      chdir(fixture_dir) do
        env = {"PATH" => "#{fixture_dir}:$PATH"}
        result = run_rux_allowing_errors("test_spec.rb", env: env)
        expect(result.exit_status).not_to eq(0)

        # TODO we gotta implement error counts
        # expect(result.out).to include("1 error")

        # Most importantly, should NOT show raw JSON parsing error
        expect(result.err).not_to include("failed to parse JSON: unexpected end of JSON input")
        expect(result.out).not_to include("failed to parse JSON: unexpected end of JSON input")
      end
    end
  end

  context "when spec directory doesn't exist" do
    it "handles missing spec directory gracefully" do
      Dir.mktmpdir do |tmpdir|
        chdir(tmpdir) do
          result = run_rux_allowing_errors

          # Should handle gracefully and show appropriate message
          expect(result.out + result.err).to include("no spec files found").or include("ERROR")
        end
      end
    end
  end
end
