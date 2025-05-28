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
      # Backup and remove existing runtime data
      @original_runtime_file = File.join(ENV["HOME"], ".cache", "rux", "runtime.json")
      @backup_file = nil
      if File.exist?(@original_runtime_file)
        @backup_file = "#{@original_runtime_file}.backup"
        FileUtils.cp(@original_runtime_file, @backup_file)
        FileUtils.rm(@original_runtime_file)
      end
    end

    after do
      # Restore original runtime data
      if @backup_file && File.exist?(@backup_file)
        FileUtils.mv(@backup_file, @original_runtime_file)
      end
      FileUtils.rm_rf(temp_cache_dir)
    end

    it "saves runtime data after running specs" do
      Dir.chdir(test_project_path) do
        # Run rux
        `#{rux_binary} -n 2 2>&1`
        expect($?.exitstatus).to eq(0)

        # Check that runtime data was saved
        runtime_file = @original_runtime_file

        expect(File.exist?(runtime_file)).to be true

        # Verify the content
        runtime_data = JSON.parse(File.read(runtime_file))
        expect(runtime_data).to be_a(Hash)
        expect(runtime_data.keys).to include("./spec/calculator_spec.rb")
        expect(runtime_data.values.all? { |v| v.is_a?(Numeric) && v > 0 }).to be true
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
        expect(File.exist?(@original_runtime_file)).to be false

        # Run with dry-run
        output = `#{rux_binary} --dry-run -n 2 2>&1`
        expect(output).to include("[dry-run] Using size-based grouped execution")
      end
    end
  end

  context "runtime-based grouping" do
    it "distributes files based on their runtime" do
      Dir.chdir(test_project_path) do
        # Create fake runtime data with uneven distribution
        cache_dir = File.join(ENV["HOME"], ".cache", "rux")
        FileUtils.mkdir_p(cache_dir)

        runtime_data = {
          "./spec/calculator_spec.rb" => 5.0,  # Very slow file
          "./spec/counter_spec.rb" => 0.1,
          "./spec/validator_spec.rb" => 0.1,
          "./spec/string_utils_spec.rb" => 0.1,
          "./spec/array_helpers_spec.rb" => 0.1,
          "./spec/date_formatter_spec.rb" => 0.1
        }

        File.write(File.join(cache_dir, "runtime.json"), JSON.pretty_generate(runtime_data))

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
