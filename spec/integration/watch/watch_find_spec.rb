require "spec_helper"

RSpec.describe "plur watch find" do
  it "explains when no watch rule matches the changed file" do
    result = run_plur_allowing_errors("watch", "find", "spec/spec_helper.rb")

    expect(result.exit_status).to eq(2)
    expect(result.out).to include("[watch] Checking spec/spec_helper.rb")
    expect(result.out).to include("[watch] No matching rule for spec/spec_helper.rb")
    expect(result.out).to include("[watch] Hint: add a [[watch]] mapping for shared files if this change should run tests.")
    expect(result.out).not_to include('msg="found rules"')
  end

  it "keeps unrelated no-rule changes terse" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("watch", "find", "README.md")

      expect(result.exit_status).to eq(2)
      expect(result.out).to include("[watch] No matching rule for README.md")
      expect(result.out).not_to include("[watch] Hint:")
    end
  end

  it "rejects one-shot dry-run flags with preview guidance" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("watch", "find", "--dry-run", "--dry-run-format=json", "lib/calculator.rb")

      expect(result).not_to be_success
      expect(result.err).to include("--dry-run does not apply to plur watch find")
      expect(result.err).to include("watch find --format=json")
      expect(result.err).to include("plur --dry-run")
    end
  end

  it "rejects inherited run and live-watch flags that do not affect watch find" do
    chdir(default_ruby_dir) do
      {
        ["watch", "find", "--workers=99", "lib/calculator.rb"] => "--workers",
        ["watch", "find", "-n", "2", "lib/calculator.rb"] => "--workers",
        ["watch", "find", "--dry-run-format=json", "lib/calculator.rb"] => "--dry-run-format",
        ["watch", "find", "--rspec-split", "lib/calculator.rb"] => "--rspec-split",
        ["watch", "find", "--first-is-1", "lib/calculator.rb"] => "--first-is-1",
        ["watch", "find", "--no-first-is-1", "lib/calculator.rb"] => "--no-first-is-1",
        ["watch", "find", "--ignore=lib/**", "lib/calculator.rb"] => "--ignore"
      }.each do |args, flag|
        result = run_plur_allowing_errors(*args)

        expect(result).not_to be_success
        expect(result.err).to include("#{flag} does not apply to plur watch find")
      end
    end
  end

  it "rejects the removed JSON file output flag as unknown" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("watch", "find", "--json=tmp/watch-find.json", "lib/calculator.rb")

      expect(result.exit_status).to eq(1)
      expect(result.err).to include("Error: --json is not a Plur flag.")
      expect(result.err).to include("plur --dry-run --dry-run-format=json [patterns...]")
      expect(result.err).to include("plur watch find --format=json <file>")
      expect(result.err).not_to include("did you mean")
    end
  end

  it "keeps human guidance when no watch mappings are configured" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("watch-find-empty-", tmp_root.to_s) do |tmpdir|
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
        run_plur_allowing_errors("watch", "find", "lib/thing.rb")
      end

      expect(result.exit_status).to eq(2)
      expect(result.out).to include("No watch mappings configured.")
      expect(result.out).to include("Either add job/watch configuration to .plur.toml")
      expect(result.err).to eq("")
    end
  end

  it "does not reject config or environment values for run-mode flags" do
    Dir.mktmpdir("watch-find-config-", ROOT_PATH.join("tmp")) do |tmpdir|
      config_path = File.join(tmpdir, "plur.toml")
      File.write(config_path, <<~TOML)
        dry-run = true
        dry-run-format = "json"
        workers = 99
        first-is-1 = false
        rspec-split = true
      TOML

      chdir(default_ruby_dir) do
        result = run_plur(
          "watch", "find", "lib/calculator.rb",
          env: {
            "PLUR_CONFIG_FILE" => config_path,
            "PLUR_RSPEC_SPLIT" => "1",
            "PARALLEL_TEST_PROCESSORS" => "7"
          }
        )

        expect(result).to be_success
        expect(result.out).to include("[watch] Would run job rspec with spec/calculator_spec.rb")
        expect(result.out).to include("[watch] Command: bundle exec rspec spec/calculator_spec.rb")
      end
    end
  end
end
