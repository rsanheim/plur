require "spec_helper"

RSpec.describe "Plur runtime tracking" do
  def runtime_cache_data
    runtime_files = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
    expect(runtime_files.size).to eq(1)
    [JSON.parse(File.read(runtime_files.first)), runtime_files.first]
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
        expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})
      end
    end
  end

  context "runtime data collection (v2 schema)" do
    around_with_tmp_plur_home

    it "writes a v2 cache with schema_version, plur_version, file aggregates, and example index" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")

        data, runtime_file = runtime_cache_data
        expect(data["schema_version"]).to eq(2)
        expect(data["plur_version"]).to be_a(String).and(satisfy { |v| !v.empty? })

        files = data["files"]
        expect(files).to be_a(Hash)
        expect(files).to include("spec/calculator_spec.rb")

        entry = files["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to be > 0
        expect(entry["mtime_unix_nano"]).to be > 0
        expect(entry["size_bytes"]).to be > 0
        expect(entry["example_index_complete"]).to be true

        examples = entry["examples"]
        expect(examples).to be_a(Hash)
        expect(examples).not_to be_empty
        sample_id, sample = examples.first
        expect(sample_id).to include("calculator_spec.rb")
        expect(sample["line_number"]).to be > 0
        expect(sample["runtime_seconds"]).to be >= 0

        expect(runtime_file).to match(%r{#{tmp_plur_home}/runtime/[a-f0-9]{8}\.json$})
      end
    end

    it "uses runtime data for grouping after the first run" do
      Dir.chdir(default_ruby_dir) do
        result = run_plur("-n", "2", "--debug")
        expect(result.err).to include("Using size-based grouping")

        result = run_plur("-n", "2", "--debug", "--dry-run")
        expect(result.err).to include("Using runtime-based grouped execution")
      end
    end

    it "ignores corrupt v2 cache files and replaces them with valid v2 JSON" do
      Dir.chdir(default_ruby_dir) do
        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path("."))[0..7]
        cache_path = File.join(runtime_dir, "#{project_hash}.json")
        File.write(cache_path, "{{{ not json")

        run_plur("-n", "2")

        data = JSON.parse(File.read(cache_path))
        expect(data["schema_version"]).to eq(2)
        expect(data["files"]).to include("spec/calculator_spec.rb")
      end
    end

    it "does not create or modify the runtime cache during --dry-run" do
      Dir.chdir(default_ruby_dir) do
        runtime_dir = File.join(tmp_plur_home, "runtime")
        run_plur("--dry-run", "-n", "2")

        matches = Dir.glob(File.join(runtime_dir, "*.json"))
        expect(matches).to be_empty
      end
    end
  end

  context "runtime-based grouping from v2 aggregates" do
    around_with_tmp_plur_home

    it "distributes files based on stored runtime_seconds" do
      Dir.chdir(default_ruby_dir) do
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path("."))[0..7]

        files = {
          "spec/calculator_spec.rb" => 5.0,
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

        cache = {
          "schema_version" => 2,
          "plur_version" => "fixture",
          "files" => files.transform_values { |rt|
            {"mtime_unix_nano" => 0, "size_bytes" => 0, "runtime_seconds" => rt,
             "example_index_complete" => false}
          }
        }

        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        File.write(File.join(runtime_dir, "#{project_hash}.json"), JSON.pretty_generate(cache))

        result = run_plur("--dry-run", "--debug", "-n", "2")
        expect(result.err).to include("Using runtime-based grouped execution")

        worker_lines = result.err.lines.select { |l| l.include?("[dry-run] Worker") }
        expect(worker_lines.size).to eq(2)

        worker0_files = worker_lines[0].scan(/spec\/[\w\/]+_spec\.rb/)
        worker1_files = worker_lines[1].scan(/spec\/[\w\/]+_spec\.rb/)

        expect(worker0_files + worker1_files).to include(match(/calculator_spec\.rb/))

        slow_worker = worker0_files.any? { |f| f.include?("calculator_spec.rb") } ? worker0_files : worker1_files
        fast_worker = (slow_worker.equal?(worker0_files) ? worker1_files : worker0_files)
        expect(slow_worker.size).to be <= fast_worker.size
      end
    end
  end

  context "aggregate-eligibility rules" do
    around_with_tmp_plur_home

    it "preserves the full-file aggregate when a focused file:line run executes a subset" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        original = initial["files"]["spec/calculator_spec.rb"]
        expect(original["runtime_seconds"]).to be > 0

        original_example_count = original["examples"].size
        focused_line = original["examples"].values.first["line_number"]

        run_plur("-n", "1", "spec/calculator_spec.rb:#{focused_line}")

        updated, _ = runtime_cache_data
        entry = updated["files"]["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to eq(original["runtime_seconds"])
        expect(entry["example_index_complete"]).to be true
        expect(entry["examples"].size).to eq(original_example_count)
      end
    end

    it "merges per-example observations by RSpec example.id without dropping others" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        examples = initial["files"]["spec/calculator_spec.rb"]["examples"]
        original_example_count = examples.size
        first_id, first_entry = examples.first

        run_plur("-n", "1", "spec/calculator_spec.rb:#{first_entry["line_number"]}")

        updated, _ = runtime_cache_data
        merged_examples = updated["files"]["spec/calculator_spec.rb"]["examples"]
        expect(merged_examples.size).to eq(original_example_count)
        expect(merged_examples).to include(first_id)
      end
    end

    it "preserves runtime data for unrun files when running a subset" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, runtime_file = runtime_cache_data
        initial_files = initial["files"]

        run_plur("-n", "1", "spec/calculator_spec.rb")

        updated = JSON.parse(File.read(runtime_file))
        updated_files = updated["files"]
        expect(updated_files.keys).to include("spec/calculator_spec.rb")
        expect(updated_files.keys.size).to eq(initial_files.keys.size)

        initial_files.each do |file, entry|
          next if file == "spec/calculator_spec.rb"
          expect(updated_files[file]["runtime_seconds"]).to eq(entry["runtime_seconds"]),
            "#{file} runtime should be preserved"
        end
      end
    end
  end
end
