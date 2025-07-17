require "spec_helper"

RSpec.describe "Plur runtime tracking" do
  context "explicit runtime dir" do
    around_with_tmp_plur_home

    it "uses RUX_HOME environment variable if provided" do
      temp_runtime_dir = File.join(tmp_plur_home, "runtime")

      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")

        expect(File.exist?(temp_runtime_dir)).to be true
        matches = Dir.glob(File.join(temp_runtime_dir, "*.json"))
        expect(matches.size).to eq(1)
        expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})
      end
    end
  end

  context "runtime data collection" do
    around_with_tmp_plur_home

    it "saves runtime data after running specs" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("-n", "2")

        runtime_file_match = result.err.match(/Runtime data saved to: (.+)/)
        expect(runtime_file_match).not_to be_nil
        runtime_file = runtime_file_match[1].strip

        expect(File.exist?(runtime_file)).to be true

        runtime_data = JSON.parse(File.read(runtime_file))
        expect(runtime_data).to be_a(Hash)
        expect(runtime_data.keys).to include("./spec/calculator_spec.rb")
        expect(runtime_data.values.all? { |v| v.is_a?(Numeric) && v > 0 }).to be true

        # Verify it's in the temp directory with a hash filename
        expect(runtime_file).to match(%r{#{tmp_plur_home}/runtime/[a-f0-9]{8}\.json$})
      end
    end

    it "uses runtime data for grouping when available" do
      chdir(default_ruby_dir) do
        result = run_plur("-n", "2")
        expect(result.err).to include("Using size-based grouped execution")

        result = run_plur("-n", "2", "--debug", "--dry-run")
        expect(result.err).to include("Using runtime-based grouped execution")
      end
    end

    it "falls back to size-based grouping when no runtime data exists" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("--dry-run", "-n", "2")
        expect(result.err).to include("[dry-run] Using size-based grouped execution")
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
        result = run_plur("--dry-run", "-n", "2", "--runtime-dir", temp_cache_dir)

        # The slow file should be in its own group or with minimal other files
        expect(result.err).to include("Using runtime-based grouped execution")

        # Extract worker assignments
        worker_lines = result.err.lines.select { |l| l.include?("[dry-run] Worker") }
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
