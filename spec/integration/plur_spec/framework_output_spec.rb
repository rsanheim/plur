require "spec_helper"

RSpec.describe "Framework output (dry-run + verbose)" do
  around_with_tmp_plur_home

  it "uses framework defaults for non-standard rspec jobs and surfaces framework in output" do
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
end
