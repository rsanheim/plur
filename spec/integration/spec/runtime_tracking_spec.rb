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
        expect(matches.first).to match(%r{#{temp_runtime_dir}/[a-f0-9]{8}\.json$})
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

    it "ignores corrupt cache files and replaces them with valid JSON" do
      Dir.chdir(default_ruby_dir) do
        runtime_dir = File.join(tmp_plur_home, "runtime")
        FileUtils.mkdir_p(runtime_dir)
        require "digest"
        project_hash = Digest::SHA256.hexdigest(File.expand_path("."))[0..7]
        cache_path = File.join(runtime_dir, "#{project_hash}.json")
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
        File.write(File.join(runtime_dir, "#{project_hash}.json"), JSON.pretty_generate(cache))

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

  context "--rspec-split (experimental)" do
    around_with_tmp_plur_home

    it "passes through file-level targets in dry-run when cache lacks examples" do
      Dir.chdir(default_ruby_dir) do
        # Without a cache, splitting is impossible.
        result = run_plur("--rspec-split", "--dry-run", "-n", "4")
        worker_lines = result.err.lines.select { |l| l.include?("[dry-run] Worker") }
        expect(worker_lines).not_to be_empty
        # No worker should have a file:line target with the splitter format.
        worker_lines.each do |line|
          line.scan(/spec\/[\w\/]+_spec\.rb(?::\d+)+/) do |target|
            raise "unexpected split target in cold-cache run: #{target}"
          end
        end
      end
    end

    it "splits long-running files into focused file:line targets after the cache is warmed" do
      Dir.chdir(default_ruby_dir) do
        # Warm the runtime cache with a real run.
        run_plur("-n", "2")
        cache, runtime_file = runtime_cache_data

        # Force calculator_spec to look slow so the splitter triggers.
        entry = cache["files"]["spec/calculator_spec.rb"]
        entry["runtime_seconds"] = 60.0
        File.write(runtime_file, JSON.pretty_generate(cache))

        result = run_plur("--rspec-split", "--dry-run", "-n", "4", "--debug")
        worker_lines = result.err.lines.select { |l| l.include?("[dry-run] Worker") }
        joined = worker_lines.join
        expect(joined).to match(%r{spec/calculator_spec\.rb(?::\d+)+}),
          "calculator_spec should appear as a file:line target"
        expect(result.err).to include("rspec-split applied")
      end
    end

    it "actually runs split file:line targets and passes" do
      Dir.chdir(default_ruby_dir) do
        # Warm cache, mark calculator as slow.
        run_plur("-n", "2")
        cache, runtime_file = runtime_cache_data
        cache["files"]["spec/calculator_spec.rb"]["runtime_seconds"] = 60.0
        File.write(runtime_file, JSON.pretty_generate(cache))

        result = run_plur("--rspec-split", "-n", "4", "spec/calculator_spec.rb")
        expect(result.exit_status).to eq(0)
      end
    end

    it "does not duplicate generated examples that share a rerunnable line" do
      Dir.chdir(project_fixture("rspec-success-simple")) do
        run_plur("-n", "2", "--color=never", "spec/generated_examples_spec.rb")
        cache, runtime_file = runtime_cache_data

        entry = cache.fetch("files").fetch("spec/generated_examples_spec.rb")
        duplicate_selector, duplicate_examples = entry.fetch("examples")
          .group_by { |example| example.fetch("location_rerun_argument") }
          .find { |_selector, examples| examples.size > 1 }
        expect(duplicate_examples.size).to eq(4)

        entry["runtime_seconds"] = 80.0
        entry.fetch("examples").each do |example|
          example["runtime_seconds"] = if example.fetch("location_rerun_argument") == duplicate_selector
            10.0
          else
            1.0
          end
        end
        File.write(runtime_file, JSON.pretty_generate(cache))

        dry_run = run_plur("--rspec-split", "--dry-run", "--debug", "-n", "4", "--color=never", "spec/generated_examples_spec.rb")
        planned_targets = dry_run.err.scan(%r{spec/generated_examples_spec\.rb(?::\d+)+})
        duplicate_line = duplicate_selector.split(":").last
        duplicate_line_occurrences = planned_targets.sum do |target|
          target.split(":").drop(1).count(duplicate_line)
        end
        expect(duplicate_line_occurrences).to eq(1)

        result = run_plur("--rspec-split", "-n", "4", "--color=never", "spec/generated_examples_spec.rb")
        expect(result.exit_status).to eq(0)
        expect(result.out).to include("8 examples, 0 failures")
      end
    end

    it "keeps shared examples grouped under their consumer rerunnable selectors" do
      Dir.chdir(project_fixture("rspec-success-simple")) do
        raw_out, _raw_err, raw_status = Open3.capture3(
          "bundle", "exec", "rspec",
          "spec/shared_example_consumers_spec.rb",
          "--format", "progress",
          "--no-color"
        )
        expect(raw_status).to be_success
        expect(raw_out).to match(/\A\s*Randomized with seed \d+\n\.\.\.\.\./)
        expect(raw_out).to include("5 examples, 0 failures")
        expect(raw_out).not_to include("with a small sum")

        warm_run = run_plur("-n", "2", "--color=never", "spec/shared_example_consumers_spec.rb")
        expect(warm_run.out).to match(/\A\.\.\.\.\./)
        expect(warm_run.out).to include("5 examples, 0 failures")
        expect(warm_run.out).not_to include("with a small sum")

        cache, runtime_file = runtime_cache_data

        files = cache.fetch("files")
        expect(files).to include("spec/shared_example_consumers_spec.rb")
        expect(files).not_to include("spec/support/shared_examples/arithmetic_examples.rb")

        entry = files.fetch("spec/shared_example_consumers_spec.rb")
        selectors = entry.fetch("examples").map { |example| example.fetch("location_rerun_argument") }
        shared_selectors = selectors.select { |selector| selector.include?("shared_example_consumers_spec.rb") }
        expect(shared_selectors.uniq.size).to be >= 2
        expect(selectors).not_to include(match(%r{spec/support/shared_examples}))

        entry["runtime_seconds"] = 80.0
        entry.fetch("examples").each do |example|
          example["runtime_seconds"] = if shared_selectors.include?(example.fetch("location_rerun_argument"))
            10.0
          else
            1.0
          end
        end
        File.write(runtime_file, JSON.pretty_generate(cache))

        dry_run = run_plur("--rspec-split", "--dry-run", "--debug", "-n", "4", "--color=never", "spec/shared_example_consumers_spec.rb")
        planned_targets = dry_run.err.scan(%r{spec/shared_example_consumers_spec\.rb(?::\d+)+})
        expect(planned_targets).not_to be_empty

        shared_selectors.each do |selector|
          line = selector.split(":").last
          occurrences = planned_targets.sum { |target| target.split(":").drop(1).count(line) }
          expect(occurrences).to eq(1)
        end

        result = run_plur("--rspec-split", "-n", "4", "--color=never", "spec/shared_example_consumers_spec.rb")
        expect(result.exit_status).to eq(0)
        expect(result.out).to match(/\A\.\.\.\.\./)
        expect(result.out).to include("5 examples, 0 failures")
        expect(result.out).not_to include("with a small sum")
      end
    end
  end
end
