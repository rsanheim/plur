require "spec_helper"

RSpec.describe "Plur --rspec-split (experimental)" do
  def runtime_cache_data
    runtime_files = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
    expect(runtime_files.size).to eq(1)
    [JSON.parse(File.read(runtime_files.first)), runtime_files.first]
  end

  around_with_tmp_plur_home

  it "preserves the cache and runs every example after a worker exits early" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("interrupted-split-", tmp_root.to_s) do |project|
      FileUtils.mkdir_p(File.join(project, "spec"))
      File.write(File.join(project, "spec", "interrupted_spec.rb"), <<~RUBY)
        RSpec.configure { |config| config.order = :defined }

        RSpec.describe "Interrupted worker" do
          it("first") { expect(1 + 1).to eq(2) }
          it("second") { expect(2 + 2).to eq(4) }
          it "third" do
            exit! 1 if ENV["PLUR_TEST_EXIT_EARLY"] == "1"
            expect(3 + 3).to eq(6)
          end
          it("fourth") { expect(4 + 4).to eq(8) }
        end
      RUBY

      Dir.chdir(project) do
        warm_run = run_plur("-n", "1", "--color=never")
        expect(warm_run.out).to include("4 examples, 0 failures")
        cache, runtime_file = runtime_cache_data
        expect(cache.fetch("files").fetch("spec/interrupted_spec.rb").fetch("examples").size).to eq(4)
        original_cache = File.binread(runtime_file)

        # Change only the environment: source freshness must still match.
        interrupted_run = run_plur_allowing_errors("-n", "1", "--color=never",
          env: {"PLUR_TEST_EXIT_EARLY" => "1"})
        expect(interrupted_run.exit_status).to eq(1)
        expect(interrupted_run.out).to include("2 examples, 0 failures")
        cache_after_interruption = File.binread(runtime_file)

        split_run = run_plur("--rspec-split", "-n", "2", "--color=never", "--debug")
        aggregate_failures do
          expect(cache_after_interruption).to eq(original_cache)
          expect(split_run.err).to include("rspec-split applied")
          expect(split_run.out).to include("4 examples, 0 failures")
        end
      end
    end
  end

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
