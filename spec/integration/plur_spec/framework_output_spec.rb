require "spec_helper"

RSpec.describe "Framework output (dry-run + verbose)" do
  around_with_tmp_plur_home

  def with_fast_rspec_project
    Dir.mktmpdir do |dir|
      spec_dir = File.join(dir, "spec")
      FileUtils.mkdir_p(spec_dir)
      File.write(File.join(spec_dir, "example_spec.rb"), <<~RUBY)
        RSpec.describe "Example" do
          it "passes" do
            expect(1).to eq(1)
          end
        end
      RUBY

      File.write(File.join(dir, ".plur.toml"), <<~TOML)
        workers = 1
        color = false
        use = "fast"

        [job.fast]
        framework = "rspec"
        cmd = ["bundle", "exec", "rspec", "--fail-fast", "{{target}}"]
        target_pattern = "spec/**/*_spec.rb"
      TOML

      yield dir
    end
  end

  def normalize_framework_dry_run(output)
    output
      .gsub(/^plur version version=.*$/, "plur version version=[VERSION]")
      .gsub(%r{-r\s+\S+/formatter/json_rows_formatter\.rb}, "-r [FORMATTER_PATH]")
      .gsub(/\bTEST_ENV_NUMBER=\d+/, "TEST_ENV_NUMBER=[N]")
  end

  def normalize_framework_snapshot(snapshot)
    snapshot.merge("stderr" => normalize_framework_dry_run(snapshot.fetch("stderr", "")))
  end

  it "uses framework defaults for non-standard rspec jobs and surfaces framework in output" do
    with_fast_rspec_project do |dir|
      chdir(dir) do
        result = run_plur("--dry-run", "--debug")

        expect(result.success?).to be(true)
        expect(result.err).to include("[dry-run] Running 1 spec [rspec] in parallel using 1 workers")
        expect(result.err).to include("framework=\"rspec\"")

        worker_line = result.err.lines.find { |line| line.include?("[dry-run] Worker 0:") }
        expect(worker_line).not_to be_nil

        expected_cmd = "bundle exec rspec --fail-fast -r #{plur_home.join("formatter", "json_rows_formatter.rb")} " \
                       "--format Plur::JsonRowsFormatter --no-color spec/example_spec.rb"
        expect(worker_line).to include(expected_cmd)
      end
    end
  end

  it "captures dry-run output contract for non-standard rspec jobs with Backspin" do
    with_fast_rspec_project do |dir|
      chdir(dir) do
        command = [plur_binary, "--dry-run"]
        result = Backspin.run(
          command,
          name: "framework_output_fast_job_dry_run",
          filter: ->(snapshot) { normalize_framework_snapshot(snapshot) }
        )

        expect(result.actual.stderr).to include("--fail-fast")
        expect(result.actual.stderr).to include("[dry-run] Running 1 spec [rspec] in parallel using 1 workers")
      end
    end
  end
end
