require "spec_helper"

RSpec.describe "Rux output performance" do
  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  describe "concurrent output handling" do
    it "produces valid output with high worker count" do
      Dir.chdir(rux_ruby_dir) do
        # Run with many workers to stress-test the output handling
        output = `#{rux_binary} -n 8 2>&1`

        # Count the dots in output
        dot_count = output.scan(".").count

        # Should have at least some dots (examples)
        expect(dot_count).to be > 0

        # Output should still be valid
        expect(output).to include("examples")
        expect(output).to include("failures")
        expect($?.exitstatus).to eq(0)
      end
    end

    it "maintains colored output when supported" do
      Dir.chdir(rux_ruby_dir) do
        # Force color output
        output = `FORCE_COLOR=1 #{rux_binary} -n 4 2>&1`

        # Should contain ANSI color codes for green dots
        expect(output).to include("\e[32m.\e[0m")
        expect($?.exitstatus).to eq(0)
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
          output = `#{rux_binary} -n 2 mixed_spec.rb 2>&1`

          # Should show both dots and F's
          expect(output).to match(/[.F]+/)

          # Should still complete successfully (exit 1 for failures)
          expect($?.exitstatus).to eq(1)
        end
      end
    end
  end
end
