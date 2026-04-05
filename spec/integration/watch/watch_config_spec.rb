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

  it "accepts [[watch]] without source and yields no matching rules in watch find" do
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

      expect(stderr).to eq("")
      expect(stdout).to include('msg="checking watch" file=lib/calculator.rb')
      expect(stdout).to include('msg="found rules" count=0')
      expect(status.exitstatus).to eq(2)
    end
  end

  it "uses the same loaded watch config in watch run and falls back to project root when source is missing", :skip_if_ci do
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

      expect(result.err).to include("Watch directories after filtering dirs=[.]")
      expect(result.success?).to be(true)
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
      expect(stdout).to include('msg="checking watch" file=lib/calculator.rb')
      expect(stdout).to include('msg="found rules" name=missing-jobs source=lib/**/*.rb jobs=[] target="[source file]"')
      expect(status.exitstatus).to eq(2)
    end
  end

  it "uses the same loaded watch config in watch run and keeps source-derived watch directories when jobs is missing", :skip_if_ci do
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

      expect(result.err).to include("Watch directories after filtering dirs=[lib]")
      expect(result.success?).to be(true)
    end
  end
end
