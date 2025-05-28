require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Rux runtime tracking" do
  let(:rux_binary) { File.expand_path("../rux/rux", __dir__) }
  let(:test_project_path) { File.join(__dir__, "..", "rux-ruby") }

  before do
    # Build rux
    Dir.chdir(File.join(__dir__, "..", "rux")) do
      system("go build -o rux", err: File::NULL)
    end
  end

  context "runtime data collection" do
    let(:temp_cache_dir) { Dir.mktmpdir }

    before do
      # Clear any existing runtime data for this project
      cache_dir = File.join(ENV["HOME"], ".cache", "rux", "runtimes")
      FileUtils.rm_rf(cache_dir) if File.exist?(cache_dir)
    end

    it "saves runtime data after running specs" do
      Dir.chdir(test_project_path) do
        # Run rux
        output = `#{rux_binary} -n 2 2>&1`
        expect($?.exitstatus).to eq(0)

        # Extract the runtime file path from output
        runtime_file_match = output.match(/Runtime data saved to: (.+)/)
        expect(runtime_file_match).not_to be_nil
        runtime_file = runtime_file_match[1].strip

        expect(File.exist?(runtime_file)).to be true

        # Verify the content
        runtime_data = JSON.parse(File.read(runtime_file))
        expect(runtime_data).to be_a(Hash)
        expect(runtime_data.keys).to include("./spec/calculator_spec.rb")
        expect(runtime_data.values.all? { |v| v.is_a?(Numeric) && v > 0 }).to be true

        # Verify it's in the runtimes subdirectory with a hash filename
        expect(runtime_file).to match(%r{\.cache/rux/runtimes/[a-f0-9]{8}\.json$})
      end
    end

    it "uses runtime data for grouping when available" do
      Dir.chdir(test_project_path) do
        # First run to generate runtime data
        `#{rux_binary} -n 2 2>&1`
        expect($?.exitstatus).to eq(0)

        # Second run should use runtime data
        output = `#{rux_binary} --dry-run -n 2 2>&1`
        expect(output).to include("[dry-run] Using runtime-based grouped execution")
      end
    end

    it "falls back to size-based grouping when no runtime data exists" do
      Dir.chdir(test_project_path) do
        # Ensure no runtime data exists (already removed in before hook)
        cache_dir = File.join(ENV["HOME"], ".cache", "rux", "runtimes")
        expect(Dir.exist?(cache_dir)).to be false

        # Run with dry-run
        output = `#{rux_binary} --dry-run -n 2 2>&1`
        expect(output).to include("[dry-run] Using size-based grouped execution")
      end
    end
  end

  context "runtime-based grouping" do
    it "distributes files based on their runtime" do
      Dir.chdir(test_project_path) do
        # First run to get the project hash
        `#{rux_binary} --dry-run -n 1 2>&1`

        # Create fake runtime data with uneven distribution
        runtimes_dir = File.join(ENV["HOME"], ".cache", "rux", "runtimes")
        FileUtils.mkdir_p(runtimes_dir)

        # Calculate project hash (simplified version - just use a known pattern)
        # We'll create multiple possible hash files to ensure one matches
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path(test_project_path))[0..7]

        runtime_data = {
          "./spec/calculator_spec.rb" => 5.0,  # Very slow file
          "./spec/counter_spec.rb" => 0.1,
          "./spec/validator_spec.rb" => 0.1,
          "./spec/string_utils_spec.rb" => 0.1,
          "./spec/array_helpers_spec.rb" => 0.1,
          "./spec/date_formatter_spec.rb" => 0.1
        }

        File.write(File.join(runtimes_dir, "#{project_hash}.json"), JSON.pretty_generate(runtime_data))

        # Run dry-run to see grouping
        output = `#{rux_binary} --dry-run -n 2 2>&1`

        # The slow file should be in its own group or with minimal other files
        expect(output).to include("Using runtime-based grouped execution")

        # Extract worker assignments
        worker_lines = output.lines.select { |l| l.include?("[dry-run] Worker") }
        expect(worker_lines.size).to eq(2)

        # Both workers should have files, but runtime distribution should be balanced
        worker0_files = worker_lines[0].scan(/spec\/[\w\/]+_spec\.rb/)
        worker1_files = worker_lines[1].scan(/spec\/[\w\/]+_spec\.rb/)

        # At least one worker should have the slow file
        expect(worker0_files + worker1_files).to include(match(/calculator_spec\.rb/))

        # The algorithm should have distributed files to balance runtime
        # The worker with calculator_spec.rb (5.0s) should have fewer other files
        if worker0_files.any? { |f| f.include?("calculator_spec.rb") }
          # Worker 0 has the slow file, so it should have fewer total files
          expect(worker0_files.size).to be <= worker1_files.size
        else
          # Worker 1 has the slow file, so it should have fewer total files
          expect(worker1_files.size).to be <= worker0_files.size
        end
      end
    end
  end
end
