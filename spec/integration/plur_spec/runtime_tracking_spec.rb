require "spec_helper"
require "digest"

RSpec.describe "Plur runtime tracking" do
  def expected_runtime_filename
    abs_path = File.expand_path(".")
    dir_name = File.basename(abs_path)
    project_hash = Digest::SHA256.hexdigest(abs_path)[0..7]
    "#{dir_name}-#{project_hash}.json"
  end

  context "explicit runtime dir" do
    around_with_tmp_plur_home

    it "uses PLUR_HOME environment variable if provided" do
      temp_runtime_dir = File.join(tmp_plur_home, "runtime")

      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")

        expect(File.exist?(temp_runtime_dir)).to be true
        matches = Dir.glob(File.join(temp_runtime_dir, "*.json"))
        expect(matches.size).to eq(1)
        expect(File.basename(matches.first)).to eq(expected_runtime_filename)
      end
    end
  end

  context "runtime data collection" do
    around_with_tmp_plur_home

    it "saves runtime data in schema v2 format after running specs" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")

        runtime_files = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
        expect(runtime_files.size).to eq(1)
        runtime_file = runtime_files.first

        runtime_data = JSON.parse(File.read(runtime_file))

        # Verify schema v2 envelope
        expect(runtime_data["schema_version"]).to eq(2)
        expect(runtime_data["project_root"]).to eq(File.expand_path("."))
        expect(runtime_data["project_hash"]).to match(/\A[a-f0-9]{8}\z/)
        expect(runtime_data["generated_at"]).to match(/\d{4}-\d{2}-\d{2}T/)
        expect(runtime_data["plur_version"]).to be_a(String)
        expect(runtime_data["framework"]).to eq("rspec")

        # Verify files map
        files = runtime_data["files"]
        expect(files).to be_a(Hash)
        expect(files.keys).to include("spec/calculator_spec.rb")

        file_entry = files["spec/calculator_spec.rb"]
        expect(file_entry["total_seconds"]).to be > 0
        expect(file_entry["example_count"]).to be > 0
        expect(file_entry["last_seen"]).to match(/\d{4}-\d{2}-\d{2}T/)

        # Verify filename is sanitized path
        expect(File.basename(runtime_file)).to eq(expected_runtime_filename)
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
        project_hash = Digest::SHA256.hexdigest(File.expand_path("."))[0..7]
        now = Time.now.utc.iso8601

        # Build a helper to create file entries
        make_entry = ->(seconds, count) {
          {"total_seconds" => seconds, "example_count" => count, "last_seen" => now}
        }

        # Create fake runtime data in schema v2 format
        runtime_data = {
          "schema_version" => 2,
          "project_root" => File.expand_path("."),
          "project_hash" => project_hash,
          "generated_at" => now,
          "plur_version" => "test",
          "framework" => "rspec",
          "files" => {
            "spec/calculator_spec.rb" => make_entry.call(5.0, 10),
            "spec/counter_spec.rb" => make_entry.call(0.1, 2),
            "spec/validator_spec.rb" => make_entry.call(0.1, 2),
            "spec/string_utils_spec.rb" => make_entry.call(0.1, 2),
            "spec/array_helpers_spec.rb" => make_entry.call(0.1, 2),
            "spec/date_formatter_spec.rb" => make_entry.call(0.1, 2),
            "spec/example_scenarios_spec.rb" => make_entry.call(0.1, 2),
            "spec/plur_ruby_spec.rb" => make_entry.call(0.1, 2),
            "spec/env_test_spec.rb" => make_entry.call(0.1, 2),
            "spec/failing_examples_spec.rb" => make_entry.call(0.1, 2),
            "spec/models/user_spec.rb" => make_entry.call(0.1, 2),
            "spec/services/email_service_spec.rb" => make_entry.call(0.1, 2)
          }
        }

        # Write fake runtime data to PLUR_HOME/runtime with sanitized filename
        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        File.write(File.join(runtime_dir, expected_runtime_filename), JSON.pretty_generate(runtime_data))

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
          expect(worker0_files.size).to be <= worker1_files.size
        else
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

        # Should still have all files in the files map, not just calculator
        expect(updated_data["schema_version"]).to eq(2)
        expect(updated_data["files"].keys).to include("spec/calculator_spec.rb")
        expect(updated_data["files"].keys.size).to eq(initial_data["files"].keys.size)

        # Other files should have their original timing preserved
        initial_data["files"].each do |file, entry|
          next if file == "spec/calculator_spec.rb"
          expect(updated_data["files"][file]["total_seconds"]).to eq(entry["total_seconds"]),
            "#{file} runtime should be preserved"
        end
      end
    end
  end
end
