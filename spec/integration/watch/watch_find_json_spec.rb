require "spec_helper"
require "json"

RSpec.describe "plur watch find JSON output" do
  it "prints structured runnable target details" do
    chdir(default_ruby_dir) do
      result = run_plur("watch", "find", "--format=json", "lib/calculator.rb")
      plan = JSON.parse(result.out)

      expect(plan).to include(
        "version" => 1,
        "mode" => "watch_find",
        "file" => "lib/calculator.rb",
        "exit_code" => 0
      )
      expect(plan["matched_rules"].first).to include(
        "name" => "lib-to-spec",
        "source" => "lib/**/*.rb",
        "jobs" => ["rspec"],
        "target" => "spec/{{match}}_spec.rb"
      )
      expect(plan["existing_targets"]).to include("rspec" => ["spec/calculator_spec.rb"])
      expect(plan["missing_targets"]).to eq({})
      expect(plan["job_plans"]).to contain_exactly(
        include(
          "job" => "rspec",
          "targets" => ["spec/calculator_spec.rb"],
          "argv" => ["bundle", "exec", "rspec", "spec/calculator_spec.rb"],
          "env" => [],
          "cwd" => default_ruby_dir.to_s,
          "shell" => "bundle exec rspec spec/calculator_spec.rb"
        )
      )
    end
  end

  it "prints structured no-rule details with exit 2" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("watch", "find", "--format=json", "spec/spec_helper.rb")
      plan = JSON.parse(result.out)

      expect(result.exit_status).to eq(2)
      expect(plan).to include(
        "version" => 1,
        "mode" => "watch_find",
        "file" => "spec/spec_helper.rb",
        "exit_code" => 2
      )
      expect(plan["matched_rules"]).to eq([])
      expect(plan["existing_targets"]).to eq({})
      expect(plan["missing_targets"]).to eq({})
      expect(plan["job_plans"]).to eq([])
    end
  end

  it "prints structured no-watch-mapping details with exit 2" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("watch-find-json-empty-", tmp_root.to_s) do |tmpdir|
      FileUtils.mkdir_p(File.join(tmpdir, "lib"))
      File.write(File.join(tmpdir, "lib", "thing.rb"), "# source\n")
      File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
        use = "custom"

        [job.custom]
        framework = "passthrough"
        cmd = ["echo", "RUN"]
        target_pattern = "lib/**/*.rb"
      TOML

      result = Dir.chdir(tmpdir) do
        run_plur_allowing_errors("watch", "find", "--format=json", "lib/thing.rb")
      end
      plan = JSON.parse(result.out)

      expect(result.exit_status).to eq(2)
      expect(result.err).to eq("")
      expect(plan).to include(
        "version" => 1,
        "mode" => "watch_find",
        "file" => "lib/thing.rb",
        "exit_code" => 2
      )
      expect(plan["matched_rules"]).to eq([])
      expect(plan["existing_targets"]).to eq({})
      expect(plan["missing_targets"]).to eq({})
      expect(plan["job_plans"]).to eq([])
    end
  end

  it "does not mask invalid explicit job selection when no watch mappings exist" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("watch-find-json-empty-use-", tmp_root.to_s) do |tmpdir|
      FileUtils.mkdir_p(File.join(tmpdir, "lib"))
      File.write(File.join(tmpdir, "lib", "thing.rb"), "# source\n")
      File.write(File.join(tmpdir, ".plur.toml"), <<~TOML)
        [job.custom]
        framework = "passthrough"
        cmd = ["echo", "RUN"]
        target_pattern = "lib/**/*.rb"
      TOML

      result = Dir.chdir(tmpdir) do
        run_plur_allowing_errors("watch", "find", "--format=json", "--use=missing", "lib/thing.rb")
      end

      expect(result.exit_status).to eq(1)
      expect(result.out).to eq("")
      expect(result.err).to include("Error: failed to select watch job: job 'missing' not found")
    end
  end
end
