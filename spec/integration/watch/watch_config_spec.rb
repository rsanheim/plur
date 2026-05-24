require "spec_helper"

RSpec.describe "plur watch config edge cases" do
  include PlurWatchHelper

  def with_temp_watch_config(toml)
    Dir.mktmpdir do |tmpdir|
      config_path = File.join(tmpdir, ".plur.toml")
      File.write(config_path, toml)
      yield config_path, tmpdir
    end
  end

  it "rejects [[watch]] without source in watch find" do
    with_temp_watch_config(<<~TOML) do |config_path, _tmpdir|
      use = "rspec"

      [[watch]]
      name = "missing-source"
      jobs = ["rspec"]
    TOML

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "lib/calculator.rb",
        chdir: default_ruby_dir.to_s
      )

      expect(stdout).to eq("")
      expect(stderr).to match(
        /configuration error in \[[^\]]+\]: watch "missing-source" has invalid source pattern "": must not be empty/
      )
      expect(status).not_to be_success
    end
  end

  it "rejects [[watch]] without source in watch run", :skip_if_ci do
    with_temp_watch_config(<<~TOML) do |config_path, tmpdir|
      use = "rspec"

      [[watch]]
      name = "missing-source"
      jobs = ["rspec"]
    TOML

      result = run_plur_watch(
        timeout: 1,
        env: {
          "PLUR_CONFIG_FILE" => config_path,
          "PLUR_HOME" => File.join(tmpdir, "plur-home")
        }
      )

      expect(result.out).to eq("")
      expect(result.err).to match(
        /configuration error in \[[^\]]+\]: watch "missing-source" has invalid source pattern "": must not be empty/
      )
      expect(result.success?).to be(false)
    end
  end

  it "accepts [[watch]] without jobs and reports jobs=[] in watch find" do
    with_temp_watch_config(<<~TOML) do |config_path, _tmpdir|
      use = "rspec"

      [[watch]]
      name = "missing-jobs"
      source = "lib/**/*.rb"
    TOML

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "lib/calculator.rb",
        chdir: default_ruby_dir.to_s
      )

      expect(stderr).to eq("")
      expect(stdout).to include("[watch] Checking lib/calculator.rb")
      expect(stdout).to include("[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)")
      expect(stdout).to include("[watch] Matched rule missing-jobs (source: lib/**/*.rb, jobs: none, target: [source file])")
      expect(stdout).to include("[watch] Would run job rspec with spec/calculator_spec.rb")
      expect(status.exitstatus).to eq(0)
    end
  end

  it "uses the same loaded watch config in watch run and merges built-in watch directories when jobs is missing", :skip_if_ci do
    with_temp_watch_config(<<~TOML) do |config_path, tmpdir|
      use = "rspec"

      [[watch]]
      name = "missing-jobs"
      source = "lib/**/*.rb"
    TOML

      result = run_plur_watch(
        timeout: 1,
        env: {
          "PLUR_CONFIG_FILE" => config_path,
          "PLUR_HOME" => File.join(tmpdir, "plur-home")
        }
      )

      expect(result.err).to include("Watch directories after filtering dirs=[lib spec]")
      expect(result.success?).to be(true)
    end
  end

  it "keeps built-in watches when user config adds a custom mapping" do
    with_temp_watch_config(<<~TOML) do |config_path, _tmpdir|
      use = "rspec"

      [[watch]]
      name = "custom-config-watch"
      source = "config/**/*.yml"
      targets = ["spec/config_spec.rb"]
      jobs = ["rspec"]
    TOML

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "lib/calculator.rb",
        chdir: default_ruby_dir.to_s
      )

      expect(stderr).to eq("")
      expect(stdout).to include("[watch] Checking lib/calculator.rb")
      expect(stdout).to include("[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/{{match}}_spec.rb)")
      expect(stdout).to include("[watch] Would run job rspec with spec/calculator_spec.rb")
      expect(status.exitstatus).to eq(0)
    end
  end

  it "lets a named user watch override a built-in watch" do
    with_temp_watch_config(<<~TOML) do |config_path, _tmpdir|
      use = "rspec"

      [[watch]]
      name = "lib-to-spec"
      source = "lib/**/*.rb"
      targets = ["spec/string_utils_spec.rb"]
      jobs = ["rspec"]
    TOML

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "lib/calculator.rb",
        chdir: default_ruby_dir.to_s
      )

      expect(stderr).to eq("")
      expect(stdout).to include("[watch] Checking lib/calculator.rb")
      expect(stdout).to include("[watch] Matched rule lib-to-spec (source: lib/**/*.rb, jobs: rspec, target: spec/string_utils_spec.rb)")
      expect(stdout).to include("[watch] Would run job rspec with spec/string_utils_spec.rb")
      expect(stdout).not_to include("spec/calculator_spec.rb")
      expect(status.exitstatus).to eq(0)
    end
  end

  it "rejects duplicate named watch mappings before merge" do
    with_temp_watch_config(<<~TOML) do |config_path, _tmpdir|
      use = "rspec"

      [[watch]]
      name = "lint"
      source = "config/**/*.yml"
      jobs = ["rspec"]

      [[watch]]
      name = "lint"
      source = "README.md"
      jobs = ["rspec"]
    TOML

      _stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "doctor",
        chdir: default_ruby_dir.to_s
      )

      expect(status).not_to be_success
      expect(stderr).to include("duplicate watch name")
      expect(stderr).to include('"lint"')
    end
  end
end
