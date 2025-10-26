require "spec_helper"
require "tempfile"
require "fileutils"
require "timeout"

RSpec.describe "plur watch integration" do
  include PlurWatchHelper

  it "starts watching the correct directories" do
    result, _streamed_out, _streamed_err = capture_watch_output

    expect(result.err).to include("plur watch starting!")
    expect(result.err).to include("plur configuration info")
    expect(result.err).to include("directories=[spec lib]")
    expect(result.err).to include("Debounce delay ms=100")
  end

  it "maps lib files to their specs" do
    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Modify a lib file
        calculator_file = default_ruby_dir.join("lib/calculator.rb")
        calculator_file.write(calculator_file.read + "\n# test")
        modified = true
      end
    end

    expect(result.err).to include("watch event=modify type=file")
    expect(result.err).to include("path=./lib/calculator.rb")
    expect(result.err).to include("plur event=run_command path=./spec/calculator_spec.rb")
  end

  it "maps nested lib files correctly", :skip_if_ci do
    # Create a nested lib file temporarily
    nested_lib = default_ruby_dir.join("lib/models/temp_model.rb")
    FileUtils.mkdir_p(File.dirname(nested_lib))

    begin
      File.write(nested_lib, "class TempModel; end")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
        # Wait for the watcher to be ready
        if !modified && err && err.include?("s/self/live@")
          File.write(nested_lib, "class TempModel; end")
          modified = true
        end
      end

      expect_file_change_logged(result.err, "./lib/models/temp_model.rb")
      expect_spec_run_logged(result.err, "./spec/models/temp_model_spec.rb")
      # It will say spec not found since we didn't create the spec
      expect(result.out).to include("file not found: spec/models/temp_model_spec.rb")
    ensure
      FileUtils.rm_f(nested_lib)
      begin
        FileUtils.rmdir(File.dirname(nested_lib))
      rescue
        nil
      end
    end
  end

  it "runs all specs when spec_helper.rb changes" do
    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
      if !modified && err && err.include?("s/self/live@")
        spec_helper = default_ruby_dir.join("spec/spec_helper.rb")
        original_content = spec_helper.read
        spec_helper.write(original_content + "\n# Modified by test")
        sleep 0.1
        spec_helper.write(original_content)
        modified = true
      end
    end

    expect_file_change_logged(result.err, "./spec/spec_helper.rb")
    # When spec_helper changes, it runs all specs
    expect_spec_run_logged(result.err, "./spec")
  end

  it "handles spec file changes" do
    $stdout.sync = true
    $stderr.sync = true

    spec_path = default_ruby_dir.join("spec/calculator_spec.rb")
    contents = spec_path.read

    modified = false
    result, _, _ = capture_watch_output(plur_timeout: 5) do |out, err|
      # Wait for watcher to be ready (live message)
      if !modified && err && err.include?("s/self/live@")
        File.write(spec_path, "# Modified\n" + contents)
        modified = true
      end
    end

    expect_file_change_logged(result.err, "./spec/calculator_spec.rb")
    expect_spec_run_logged(result.err, "./spec/calculator_spec.rb")
  ensure
    # Restore original content
    File.write(spec_path, contents) if contents
  end

  it "ignores non-Ruby files" do
    readme_file = default_ruby_dir.join("README.md")
    original_content = readme_file.read

    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Modify README by appending content
        readme_file.write(original_content + "\n<!-- test -->")
        modified = true
      end
    end

    output = result.out + result.err

    # Should not see any change events for README.md
    expect(output).not_to include("Changed: README.md")
  ensure
    readme_file.write(original_content) if original_content
  end

  it "respects custom debounce delay" do
    result, _streamed_out, _streamed_err = capture_watch_output(plur_timeout: 2, debounce: 250)

    output = result.out + result.err

    expect(output).to include("Debounce delay ms=250")
  end

  it "handles multiple file changes with debouncing" do
    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output(debounce: 200) do |out, err|
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Rapidly modify multiple files
        calc_file = default_ruby_dir.join("lib/calculator.rb")
        calc_file.write(calc_file.read + "\n# test")

        str_file = default_ruby_dir.join("lib/string_utils.rb")
        str_file.write(str_file.read + "\n# test")

        val_file = default_ruby_dir.join("lib/validator.rb")
        val_file.write(val_file.read + "\n# test")

        modified = true
      end
    end

    # Should see all changes in stderr
    expect_file_change_logged(result.err, "./lib/calculator.rb")
    expect_file_change_logged(result.err, "./lib/string_utils.rb")
    expect_file_change_logged(result.err, "./lib/validator.rb")

    # Should run corresponding specs
    expect_spec_run_logged(result.err, "./spec/calculator_spec.rb")
    expect_spec_run_logged(result.err, "./spec/string_utils_spec.rb")
    expect_spec_run_logged(result.err, "./spec/validator_spec.rb")
  end

  describe "Rails-style mappings", :skip_if_ci do
    let(:app_dir) { File.join(default_ruby_dir, "app") }
    let(:plur_timeout) { ENV["CI"] ? 10 : 2 }

    before do
      # Create Rails-like directory structure
      FileUtils.mkdir_p(File.join(app_dir, "models"))
      FileUtils.mkdir_p(File.join(app_dir, "controllers"))
    end

    after do
      FileUtils.rm_rf(app_dir)
    end

    it "detects app directory and watches it" do
      result, _streamed_out, _streamed_err = capture_watch_output(plur_timeout: 2)

      # Check that app directory is being watched
      expect(result.err).to include("directories=[spec lib app]")
    end

    it "maps app/models files to spec/models" do
      model_file = File.join(app_dir, "models/user.rb")
      File.write(model_file, "class User; end")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output(plur_timeout: plur_timeout) do |out, err|
        if !modified && err && err.include?("s/self/live@")
          File.write(model_file, "class User\n  # Modified\nend")
          modified = true
        end
      end

      # Check file change and spec run in stderr
      expect_file_change_logged(result.err, "./app/models/user.rb")
      expect_spec_run_logged(result.err, "./spec/models/user_spec.rb")
    end

    it "maps app/controllers files to spec/controllers" do
      controller_file = File.join(app_dir, "controllers/users_controller.rb")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output(plur_timeout: plur_timeout) do |out, err|
        if !modified && err && err.include?("s/self/live@")
          File.write(controller_file, "class UsersController\n  def index\n    # Modified\n  end\nend")
          modified = true
        end
      end

      expect_file_change_logged(result.err, "./app/controllers/users_controller.rb")
      expect_spec_run_logged(result.err, "./spec/controllers/users_controller_spec.rb")
    end
  end
end
