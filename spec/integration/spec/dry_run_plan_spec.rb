require "spec_helper"
require "json"

RSpec.describe "dry-run JSON plan output" do
  it "prints a structured one-shot plan to stdout" do
    chdir(default_ruby_dir) do
      result = run_plur("--dry-run", "--dry-run-format=json", "-n", "2", "spec/calculator_spec.rb", "spec/string_utils_spec.rb")

      plan = JSON.parse(result.out)

      expect(plan).to include(
        "version" => 1,
        "mode" => "spec",
        "targets" => contain_exactly("spec/calculator_spec.rb", "spec/string_utils_spec.rb"),
        "warnings" => []
      )
      expect(plan["job"]).to include(
        "name" => "rspec",
        "framework" => "rspec",
        "reason" => "explicit_patterns"
      )
      expect(plan["workers"].size).to eq(2)
      expect(plan["workers"].first).to include("index", "targets", "argv", "env", "shell")
      expect(result.err).not_to include("[dry-run] Worker")
    end
  end

  it "includes warnings in the structured plan" do
    chdir(default_ruby_dir) do
      File.write("spec/helper.rb", "# helper file")

      begin
        result = run_plur("--dry-run", "--dry-run-format=json", "spec/helper.rb")
        plan = JSON.parse(result.out)

        expect(plan["warnings"]).to include("target 'spec/helper.rb' does not match selected job 'rspec' target pattern 'spec/**/*_spec.rb'")
      ensure
        File.delete("spec/helper.rb") if File.exist?("spec/helper.rb")
      end
    end
  end

  it "includes configured job env in worker plans without dumping inherited env" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("dry-run-json-env-", tmp_root.to_s) do |tmpdir|
      FileUtils.mkdir_p(File.join(tmpdir, "spec"))
      File.write(File.join(tmpdir, "spec", "env_spec.rb"), "# target placeholder\n")
      File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
        use = "custom"

        [job.custom]
        framework = "rspec"
        cmd = ["bin/rspec"]
        env = ["CUSTOM_TOKEN=secret"]
        target_pattern = "spec/**/*_spec.rb"
      TOML

      result = Dir.chdir(tmpdir) do
        run_plur("--dry-run", "--dry-run-format=json")
      end
      plan = JSON.parse(result.out)
      worker = plan.fetch("workers").first

      expect(worker.fetch("env")).to include("CUSTOM_TOKEN=secret")
      expect(worker.fetch("env")).not_to include(a_string_starting_with("PATH="))
      expect(worker.fetch("shell")).to include("CUSTOM_TOKEN=secret")
    end
  end

  it "requires dry-run when requesting JSON dry-run format" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("--dry-run-format=json", "spec/calculator_spec.rb")

      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("--dry-run-format requires --dry-run")
    end
  end
end
