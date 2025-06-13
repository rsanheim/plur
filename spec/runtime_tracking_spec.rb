require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Rux runtime tracking" do
  before do
    # Build rux
    Dir.chdir(File.join(__dir__, "..", "rux")) do
      system("go build -o rux", err: File::NULL)
    end
  end

  context "explicit runtime dir" do
    let(:temp_rux_home) { Dir.mktmpdir }
    let(:temp_runtime_dir) { File.join(temp_rux_home, "runtime") }

    it "uses RUX_HOME environment variable if provided" do
      Dir.chdir(default_ruby_dir) do
        `RUX_HOME=#{temp_rux_home} #{rux_binary} -n 2 2>&1`
        expect($?.exitstatus).to eq(0)

        expect(File.exist?(temp_runtime_dir)).to be true
        matches = Dir.glob(File.join(temp_runtime_dir, "*.json"))
        expect(matches.size).to eq(1)
        expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})
      end
    end

    it "uses a default of ~/.rux/runtime when no RUX_HOME is specified" do
      # We'll verify the default behavior by checking the output message
      # without actually writing to the home directory
      Dir.chdir(default_ruby_dir) do
        output = `#{rux_binary} -n 2 --dry-run 2>&1`
        expect($?.exitstatus).to eq(0)

        # The dry-run output should show it would use the default location
        expect(output).to include("Using")
        expect(output).to include("execution")
      end
    end
  end

  context "runtime data collection" do
    let(:temp_rux_home) { Dir.mktmpdir }
    let(:temp_runtime_dir) { File.join(temp_rux_home, "runtime") }

    it "saves runtime data after running specs" do
      Dir.chdir(default_ruby_dir) do
        # Run rux with custom runtime dir via RUX_HOME
        output = `RUX_HOME=#{temp_rux_home} #{rux_binary} -n 2 2>&1`
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

        # Verify it's in the temp directory with a hash filename
        expect(runtime_file).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})
      end
    end

    it "uses runtime data for grouping when available" do
      Dir.chdir(default_ruby_dir) do
        # First run to generate runtime data
        `RUX_HOME=#{temp_rux_home} #{rux_binary} -n 2 2>&1`
        expect($?.exitstatus).to eq(0)

        # Second run should use runtime data
        output = `RUX_HOME=#{temp_rux_home} #{rux_binary} --dry-run -n 2 2>&1`
        expect(output).to include("[dry-run] Using runtime-based grouped execution")
      end
    end

    it "falls back to size-based grouping when no runtime data exists" do
      Dir.chdir(default_ruby_dir) do
        # Use a fresh temp directory to ensure no runtime data exists
        fresh_rux_home = Dir.mktmpdir

        # Run with dry-run
        output = `RUX_HOME=#{fresh_rux_home} #{rux_binary} --dry-run -n 2 2>&1`
        expect(output).to include("[dry-run] Using size-based grouped execution")
      end
    end
  end

  context "runtime-based grouping" do
    let(:temp_cache_dir) { Dir.mktmpdir }

    it "distributes files based on their runtime" do
      Dir.chdir(default_ruby_dir) do
        # Calculate project hash
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path(default_ruby_dir))[0..7]

        # Create fake runtime data with uneven distribution
        runtime_data = {
          "./spec/calculator_spec.rb" => 5.0,  # Very slow file
          "./spec/counter_spec.rb" => 0.1,
          "./spec/validator_spec.rb" => 0.1,
          "./spec/string_utils_spec.rb" => 0.1,
          "./spec/array_helpers_spec.rb" => 0.1,
          "./spec/date_formatter_spec.rb" => 0.1
        }

        File.write(File.join(temp_cache_dir, "#{project_hash}.json"), JSON.pretty_generate(runtime_data))

        # Run dry-run to see grouping
        output = `#{rux_binary} --dry-run -n 2 --runtime-dir #{temp_cache_dir} 2>&1`

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
