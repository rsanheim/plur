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
        run_plur("-n", "2")

        # Find the runtime file directly instead of parsing logs
        runtime_files = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
        expect(runtime_files.size).to eq(1)
        runtime_file = runtime_files.first

        runtime_data = JSON.parse(File.read(runtime_file))
        expect(runtime_data).to be_a(Hash)
        # Paths are saved without "./" prefix (matches glob discovery)
        expect(runtime_data.keys).to include("spec/calculator_spec.rb")
        expect(runtime_data.values.all? { |v| v.is_a?(Numeric) && v > 0 }).to be true

        # Verify it's in the temp directory with a hash filename
        expect(runtime_file).to match(%r{#{tmp_plur_home}/runtime/[a-f0-9]{8}\.json$})
      end
    end

    it "uses runtime data for grouping when available" do
      chdir(default_ruby_dir) do
        # First run creates runtime data (use --debug to see log messages)
        result = run_plur("-n", "2", "--debug")
        expect(result.err).to include("Using size-based grouping")

        # Second run should use runtime data
        result = run_plur("-n", "2", "--debug", "--dry-run")
        expect(result.err).to include("Using runtime-based grouped execution")
      end
    end
  end

  context "runtime-based grouping" do
    around_with_tmp_plur_home

    it "distributes files based on their runtime" do
      Dir.chdir(default_ruby_dir) do
        # Calculate project hash
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path("."))[0..7]

        # Create fake runtime data with uneven distribution
        # Paths should NOT have "./" prefix (matches glob discovery format)
        # Must match actual file paths including subdirectories
        runtime_data = {
          "spec/calculator_spec.rb" => 5.0,  # Very slow file
          "spec/counter_spec.rb" => 0.1,
          "spec/validator_spec.rb" => 0.1,
          "spec/string_utils_spec.rb" => 0.1,
          "spec/array_helpers_spec.rb" => 0.1,
          "spec/date_formatter_spec.rb" => 0.1,
          "spec/example_scenarios_spec.rb" => 0.1,
          "spec/plur_ruby_spec.rb" => 0.1,
          "spec/env_test_spec.rb" => 0.1,
          "spec/failing_examples_spec.rb" => 0.1,
          "spec/models/user_spec.rb" => 0.1,
          "spec/services/email_service_spec.rb" => 0.1
        }

        # Write fake runtime data to PLUR_HOME/runtime
        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        File.write(File.join(runtime_dir, "#{project_hash}.json"), JSON.pretty_generate(runtime_data))

        # Run dry-run with debug to see grouping strategy
        result = run_plur("--dry-run", "--debug", "-n", "2")

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

  context "merge behavior" do
    around_with_tmp_plur_home

    it "preserves runtime data for unrun files when running a subset" do
      Dir.chdir(default_ruby_dir) do
        # Run all specs to generate initial runtime data
        run_plur("-n", "2")

        # Read the initial runtime data
        runtime_files = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
        expect(runtime_files.size).to eq(1)
        runtime_file = runtime_files.first
        initial_data = JSON.parse(File.read(runtime_file))

        # Run only a subset of specs
        run_plur("-n", "1", "spec/calculator_spec.rb")

        # Read the updated runtime data
        updated_data = JSON.parse(File.read(runtime_file))

        # Should still have all files, not just calculator
        expect(updated_data.keys).to include("spec/calculator_spec.rb")
        expect(updated_data.keys.size).to eq(initial_data.keys.size)

        # Other files should have their original timing preserved
        initial_data.each do |file, time|
          next if file == "spec/calculator_spec.rb" # This one may have changed
          expect(updated_data[file]).to eq(time), "#{file} runtime should be preserved"
        end
      end
    end
  end
end
