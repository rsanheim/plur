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

  it "rejects [[watch]] without source" do
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
      expect(stderr).to include('watch "missing-source" must define source')
      expect(status).not_to be_success
    end
  end

  it "rejects [[watch]] without jobs" do
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

      expect(stdout).to eq("")
      expect(stderr).to include('watch "missing-jobs" must define at least one job')
      expect(status).not_to be_success
    end
  end

  it "runs no-target watch jobs without passing the changed file" do
    with_temp_watch_config(<<~TOML) do |config_path, tmpdir|
      use = "build"

      [job.build]
      cmd = ["echo", "build"]

      [[watch]]
      name = "go-build"
      source = "**/*.go"
      no_targets = true
      jobs = ["build"]
    TOML

      File.write(File.join(tmpdir, "runner.go"), "package main\n")

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "runner.go",
        chdir: tmpdir
      )

      expect(stderr).to eq("")
      expect(stdout).to include('msg="checking watch" file=runner.go')
      expect(stdout).to include('msg="found rules" name=go-build source=**/*.go jobs=[build] target="[no targets]"')
      expect(stdout).not_to include("files=runner.go")
      expect(status.exitstatus).to eq(0)
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
      expect(stdout).to include('msg="checking watch" file=lib/calculator.rb')
      expect(stdout).to include('msg="found rules" name=lib-to-spec source=lib/**/*.rb')
      expect(stdout).to include('msg="found files" files=spec/calculator_spec.rb')
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
      expect(stdout).to include('msg="checking watch" file=lib/calculator.rb')
      expect(stdout).to include('msg="found rules" name=lib-to-spec source=lib/**/*.rb jobs=[rspec] target=spec/string_utils_spec.rb')
      expect(stdout).to include('msg="found files" files=spec/string_utils_spec.rb')
      expect(stdout).not_to include("spec/calculator_spec.rb")
      expect(status.exitstatus).to eq(0)
    end
  end

  it "shows no watch mappings in an empty project without a detectable framework" do
    Dir.mktmpdir do |tmpdir|
      stdout, stderr, status = Open3.capture3(
        {"HOME" => tmpdir},
        plur_binary, "watch", "find", "lib/foo.rb",
        chdir: tmpdir
      )

      expect(stderr).to eq("")
      expect(stdout).to include("No watch mappings configured.")
      expect(status.exitstatus).to eq(0)
    end
  end

  it "finds custom watch rules when config has no use key" do
    with_temp_watch_config(<<~TOML) do |config_path, tmpdir|
      [job.build]
      cmd = ["echo", "build"]

      [[watch]]
      name = "go-build"
      source = "**/*.go"
      no_targets = true
      jobs = ["build"]
    TOML

      File.write(File.join(tmpdir, "runner.go"), "package main\n")

      stdout, stderr, status = Open3.capture3(
        {"PLUR_CONFIG_FILE" => config_path},
        plur_binary, "watch", "find", "runner.go",
        chdir: tmpdir
      )

      expect(stderr).to eq("")
      expect(stdout).to include('msg="found rules" name=go-build source=**/*.go jobs=[build] target="[no targets]"')
      expect(stdout).to include('msg="would run" job=build cmd="echo build"')
      expect(status.exitstatus).to eq(0)
    end
  end

  it "renders built-in Go watch targets as local package patterns" do
    tmp_root = ROOT_PATH.join("tmp")
    FileUtils.mkdir_p(tmp_root)

    Dir.mktmpdir("go-watch-defaults-", tmp_root.to_s) do |tmpdir|
      FileUtils.mkdir_p(File.join(tmpdir, "pkg"))
      File.write(File.join(tmpdir, "go.mod"), "module example.test/plur-watch\n")
      File.write(File.join(tmpdir, "pkg", "runner.go"), "package pkg\n")

      stdout, stderr, status = Open3.capture3(
        {"HOME" => tmpdir},
        plur_binary, "-u", "go-test", "watch", "find", "pkg/runner.go",
        chdir: tmpdir
      )

      expect(stderr).to eq("")
      expect(stdout).to include('msg="found rules" name=go-source source=**/*.go jobs=[go-test] target=./{{dir_relative}}')
      expect(stdout).to include('msg="found files" files=./pkg/')
      expect(stdout).to include('msg="would run" job=go-test cmd="go test ./pkg/"')
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
