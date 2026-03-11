require "spec_helper"
require "tempfile"
require "fileutils"
require "timeout"

RSpec.describe "plur watch integration" do
  include PlurWatchHelper

  it "starts watching the correct directories" do
    result = run_plur_watch(until_output: "directories=[lib spec]")

    expect(result.err).to include("plur watch starting!")
    expect(result.err).to include("Watch configuration loaded")
    expect(result.err).to include("directories=[lib spec]")
    expect(result.err).to include("Debounce delay ms=30")
  end

  fit "user configured watches are additive to the default watches, and do not replace them" do
    Pathname.mktmpdir do |tmpdir|
      config_path = tmpdir.join(".plur.toml")
      File.write(config_path, <<~TOML)
        [[watch]]
        source = "config/initializers/**/*.rb"
        targets = ["spec/initializers/{{match}}_spec.rb"]
        jobs = ["rspec"]
      TOML

      FileUtils.cp_r(default_rails_dir, tmpdir)

      app_dir = tmpdir.join("default-rails")
      user_model = app_dir.join("app/models/user.rb")
      initializer = app_dir.join("config/initializers/whatever.rb")
      initializer_spec = app_dir.join("spec/initializers/whatever_spec.rb")

      initializer_spec_msg = %|[plur]z bundle exec rspec spec/initializers/whatever_spec.rb|
      user_spec_msg = %|[plur] bundle exec rspec spec/models/user_spec.rb|

      system("bundle install", chdir: app_dir, exception: true)

      env = {"PLUR_CONFIG_FILE" => config_path.to_s}
      result = run_plur_watch(dir: app_dir, timeout: 10, env: env, until_output: initializer_spec_msg) do
        user_model.write(user_model.read + "\n# Modified by test")

        initializer.write("class Whatever\nend")
        sleep 0.5
        puts initializer.read
        initializer_spec.parent.mkpath
        initializer_spec.write("require 'whatever'\nRSpec.describe Whatever do\n  it 'works' do\n    expect(Whatever.new).to be_truthy\n  end\nend")
      end
      pp result.err
      pp result.out
      expect(result.out).to include(initializer_spec_msg)
      expect(result.out).to include(user_spec_msg)
  end

  end

  it "deduplicates overlapping watch directories to avoid duplicate runs" do
    calculator_file = default_ruby_dir.join("lib/calculator.rb")
    original_content = calculator_file.read

    Dir.mktmpdir do |tmpdir|
      config_path = File.join(tmpdir, ".plur.toml")
      File.write(config_path, <<~TOML)
        use = "rspec"

        [[watch]]
        source = "**/*_spec.rb"
        jobs = ["rspec"]

        [[watch]]
        source = "lib/**/*.rb"
        targets = ["spec/{{match}}_spec.rb"]
        jobs = ["rspec"]
      TOML

      plur_home = File.join(tmpdir, "plur-home")
      result = run_plur_watch(
        timeout: 10,
        env: {"PLUR_CONFIG_FILE" => config_path, "PLUR_HOME" => plur_home},
        until_output: "Executing job"
      ) do
        calculator_file.write(original_content + "\n# Modified by test")
      end

      directories_line = result.err.lines.find { |line| line.include?("directories=") }
      expect(directories_line).not_to be_nil

      raw_directories = directories_line[/directories=\[([^\]]*)\]/, 1].to_s
      directories = raw_directories.split(" ").reject(&:empty?)
      expect(directories.size).to eq(1), "Expected 1 watch directory, got #{directories.inspect}"

      executions = result.err.scan('Executing job job="rspec"').count
      expect(executions).to eq(1), "Expected 1 job execution, got #{executions}"
    ensure
      calculator_file.write(original_content)
    end
  end

  it "detects lib file changes" do
    calculator_file = default_ruby_dir.join("lib/calculator.rb")
    original_content = calculator_file.read

    result = run_plur_watch(timeout: 10, until_output: "Executing job") do
      calculator_file.write(original_content + "\n# test")
    end

    expect(result.err).to include('event="modify" type="file"')
    expect(result.err).to include('path="lib/calculator.rb"')
    expect(result.err).to include('Executing job job="rspec"')
  ensure
    calculator_file.write(original_content) if original_content
  end

  it "detects spec_helper.rb changes" do
    spec_helper = default_ruby_dir.join("spec/spec_helper.rb")
    original_content = spec_helper.read

    result = run_plur_watch(timeout: 10, until_output: "No existing targets") do
      spec_helper.write(original_content + "\n# Modified by test")
    end

    expect_file_change_logged(result.err, "spec/spec_helper.rb")
    # spec_helper.rb changes are detected but don't trigger jobs (no matching targets)
    expect(result.err).to include("No existing targets for file")
  ensure
    spec_helper.write(original_content) if original_content
  end

  it "detects spec file changes" do
    spec_path = default_ruby_dir.join("spec/calculator_spec.rb")
    contents = spec_path.read

    result = run_plur_watch(timeout: 10, until_output: "Executing job") do
      File.write(spec_path, "# Modified\n" + contents)
    end

    expect_file_change_logged(result.err, "spec/calculator_spec.rb")
    # Watch now maps file changes to jobs and executes them
    expect(result.err).to include("Executing job")
  ensure
    File.write(spec_path, contents) if contents
  end

  it "ignores non-Ruby files" do
    readme_file = default_ruby_dir.join("README.md")
    original_content = readme_file.read

    result = run_plur_watch(timeout: 3) do
      readme_file.write(original_content + "\n<!-- test -->")
    end

    output = result.out + result.err

    # Should not see any change events for README.md
    expect(output).not_to include("Changed: README.md")
  ensure
    readme_file.write(original_content) if original_content
  end

  it "respects custom debounce delay" do
    result = run_plur_watch(timeout: 10, debounce: 250, until_output: "Debounce delay ms=250")

    output = result.out + result.err

    expect(output).to include("Debounce delay ms=250")
  end

  it "detects multiple file changes with debouncing" do
    result = run_plur_watch(timeout: 10, debounce: 200, until_output: "Executing job") do
      calc_file = default_ruby_dir.join("lib/calculator.rb")
      calc_file.write(calc_file.read + "\n# test")

      str_file = default_ruby_dir.join("lib/string_utils.rb")
      str_file.write(str_file.read + "\n# test")

      val_file = default_ruby_dir.join("lib/validator.rb")
      val_file.write(val_file.read + "\n# test")
    end

    # Should see all changes in stderr
    expect_file_change_logged(result.err, "lib/calculator.rb")
    expect_file_change_logged(result.err, "lib/string_utils.rb")
    expect_file_change_logged(result.err, "lib/validator.rb")
    expect(result.err).to include("Executing job")
  end
end
