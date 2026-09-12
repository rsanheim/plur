require "spec_helper"
require "time"

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
        expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{16}\.json$})
      end
    end
  end

  context "runtime data collection (versioned schema)" do
    around_with_tmp_plur_home

    it "writes a cache with meta, run metadata, file aggregates, and example index" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")

        data, runtime_file = runtime_cache_data
        expect(data["meta"]["schema_version"]).to eq(4)
        expect(data["meta"]["plur_version"]).to be_a(String).and(satisfy { |v| !v.empty? })
        expect(data["run"]["cwd"]).to eq(default_ruby_dir.to_s)
        expect(Time.iso8601(data["run"]["last_run_at"]).utc.iso8601).to eq(data["run"]["last_run_at"])

        files = data["files"]
        expect(files).to be_a(Hash)
        expect(files).to include("spec/calculator_spec.rb")

        entry = files["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to be > 0
        expect(entry["mtime_unix_nano"]).to be > 0
        expect(entry["size_bytes"]).to be > 0
        expect(entry).not_to include("example_index_complete")

        examples = entry["examples"]
        expect(examples).to be_a(Array)
        expect(examples).not_to be_empty
        sample = examples.first
        sample_id = sample["id"]
        expect(sample_id).to include("calculator_spec.rb")
        expect(sample["line_number"]).to be > 0
        expect(sample["runtime_seconds"]).to be >= 0

        expect(runtime_file).to match(%r{#{tmp_plur_home}/runtime/[a-f0-9]{16}\.json$})
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

    it "ignores corrupt cache files and replaces them with valid JSON" do
      Dir.chdir(default_ruby_dir) do
        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        cache_path = run_plur("doctor").out[/^Runtime Data:\s+(.+)$/, 1]
        expect(cache_path).not_to be_nil
        File.write(cache_path, "{{{ not json")

        run_plur("-n", "2")

        data = JSON.parse(File.read(cache_path))
        expect(data["meta"]["schema_version"]).to eq(4)
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

  context "runtime-based grouping from cache aggregates" do
    around_with_tmp_plur_home

    it "distributes files based on stored runtime_seconds" do
      Dir.chdir(default_ruby_dir) do
        cache_path = run_plur("doctor").out[/^Runtime Data:\s+(.+)$/, 1]
        expect(cache_path).not_to be_nil

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
          "meta" => {
            "schema_version" => 4,
            "plur_version" => "fixture"
          },
          "run" => {
            "cwd" => default_ruby_dir,
            "last_run_at" => "2026-05-22T00:00:00Z"
          },
          "files" => files.transform_values { |rt|
            {"mtime_unix_nano" => 0, "size_bytes" => 0, "runtime_seconds" => rt}
          }
        }

        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        File.write(cache_path, JSON.pretty_generate(cache))

        result = run_plur("--dry-run", "--debug", "-n", "2")
        expect(result.err).to include("Using runtime-based grouped execution")

        worker_lines = result.err.lines.select { |l| l.include?("[dry-run] Worker") }
        expect(worker_lines.size).to eq(2)

        worker0_files = worker_lines[0].scan(/spec\/[\w\/]+_spec\.rb/)
        worker1_files = worker_lines[1].scan(/spec\/[\w\/]+_spec\.rb/)

        expect(worker0_files + worker1_files).to include(match(/calculator_spec\.rb/))

        slow_worker = (worker0_files.any? { |f| f.include?("calculator_spec.rb") }) ? worker0_files : worker1_files
        fast_worker = slow_worker.equal?(worker0_files) ? worker1_files : worker0_files
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
        focused_line = original["examples"].first["line_number"]

        run_plur("-n", "1", "spec/calculator_spec.rb:#{focused_line}")

        updated, _ = runtime_cache_data
        entry = updated["files"]["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to eq(original["runtime_seconds"])
        expect(entry).not_to include("example_index_complete")
        expect(entry["examples"].size).to eq(original_example_count)
      end
    end

    it "merges per-example observations by RSpec example.id without dropping others" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        examples = initial["files"]["spec/calculator_spec.rb"]["examples"]
        original_example_count = examples.size
        first_entry = examples.first
        first_id = first_entry.fetch("id")

        run_plur("-n", "1", "spec/calculator_spec.rb:#{first_entry["line_number"]}")

        updated, _ = runtime_cache_data
        merged_examples = updated["files"]["spec/calculator_spec.rb"]["examples"]
        expect(merged_examples.size).to eq(original_example_count)
        expect(merged_examples.map { |example| example.fetch("id") }).to include(first_id)
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

    it "does not update default aggregates when --tag is supplied" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        original = initial["files"]["spec/calculator_spec.rb"]

        run_plur_allowing_errors("-n", "2", "--tag=focus")

        updated, _ = runtime_cache_data
        entry = updated["files"]["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to eq(original["runtime_seconds"]),
          "tagged runs are classified as partial; aggregates must not change"
      end
    end

    it "does not update default aggregates when fail-fast aborts the run" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        original = initial["files"]["spec/calculator_spec.rb"]

        run_plur_allowing_errors("-n", "1", "--", "--fail-fast")

        updated, _ = runtime_cache_data
        entry = updated["files"]["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to eq(original["runtime_seconds"]),
          "fail-fast/aborted runs must not overwrite the aggregate"
      end
    end

    it "falls back to file-level grouping when arbitrary passthrough args are present" do
      Dir.chdir(default_ruby_dir) do
        run_plur("-n", "2")
        initial, _ = runtime_cache_data
        original = initial["files"]["spec/calculator_spec.rb"]

        run_plur("-n", "2", "--", "--seed", "1234")

        updated, _ = runtime_cache_data
        entry = updated["files"]["spec/calculator_spec.rb"]
        expect(entry["runtime_seconds"]).to eq(original["runtime_seconds"]),
          "any passthrough arg makes the run partial"
      end
    end
  end
end
