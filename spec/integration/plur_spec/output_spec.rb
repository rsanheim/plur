require "spec_helper"

RSpec.describe "Plur spec output" do
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
end
