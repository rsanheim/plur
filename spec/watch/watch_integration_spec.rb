require "spec_helper"
require "tempfile"
require "fileutils"
require "timeout"

RSpec.describe "rux watch integration" do
  include RuxWatchHelper

  it "starts watching the correct directories" do
    result, _streamed_out, _streamed_err = capture_watch_output

    expect(result.out).to include("Starting rux watch mode")
    expect(result.out).to include("Watching directories: spec, lib")
    expect(result.err).to include("Debounce delay ms=100")
  end

  it "maps lib files to their specs" do
    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Modify a lib file
        calculator_file = rux_ruby_dir.join("lib/calculator.rb")
        calculator_file.write(calculator_file.read + "\n# test")
        modified = true
      end
    end

    expect(result.err).to include("Changed: lib/calculator.rb")
    expect(result.err).to include("Running: spec/calculator_spec.rb")
  end

  it "maps nested lib files correctly" do
    # Create a nested lib file temporarily
    nested_lib = rux_ruby_dir.join("lib/models/temp_model.rb")
    FileUtils.mkdir_p(File.dirname(nested_lib))

    begin
      File.write(nested_lib, "class TempModel; end")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
        # Wait for the watcher to be ready
        if !modified && err && err.include?("s/self/live@")
          # Write to trigger a modify event
          File.write(nested_lib, "class TempModel; end")
          modified = true
        end
      end

      # Combine stdout and stderr for tests that check both
      output = result.out + result.err

      expect(output).to include("Changed: lib/models/temp_model.rb")
      expect(output).to include("Running: spec/models/temp_model_spec.rb")
      # It will say spec not found since we didn't create the spec
      expect(output).to include("Spec file not found: spec/models/temp_model_spec.rb")
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
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Modify spec_helper.rb by appending a comment
        spec_helper = rux_ruby_dir.join("spec/spec_helper.rb")
        original_content = spec_helper.read
        spec_helper.write(original_content + "\n# Modified by test")
        # Restore original content after a moment
        sleep 0.1
        spec_helper.write(original_content)
        modified = true
      end
    end

    expect(result.err).to include("Changed: spec/spec_helper.rb")
    expect(result.err).to include("Running: spec")
  end

  it "handles spec file changes" do
    $stdout.sync = true
    $stderr.sync = true

    spec_path = rux_ruby_dir.join("spec/calculator_spec.rb")
    contents = spec_path.read

    modified = false
    result, streamed_out, streamed_err = capture_watch_output(rux_timeout: 5) do |out, err|
      # Wait for watcher to be ready (live message)
      if !modified && err && err.include?("s/self/live@")
        File.write(spec_path, "# Modified\n" + contents)
        modified = true
      end
    end

    pp ["command result", result]
    pp ["streamed out", streamed_out]
    pp ["streamed err", streamed_err]
    expect(result.err).to include("Changed: spec/calculator_spec.rb")
    expect(result.err).to include("Running: spec/calculator_spec.rb")
  ensure
    # Restore original content
    File.write(spec_path, contents) if contents
  end

  it "ignores non-Ruby files" do
    readme_file = rux_ruby_dir.join("README.md")
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
    # Restore original content
    readme_file.write(original_content) if original_content
  end

  it "respects custom debounce delay" do
    result, _streamed_out, _streamed_err = capture_watch_output(rux_timeout: 2, debounce: 250)

    output = result.out + result.err

    expect(output).to include("Debounce delay ms=250")
  end

  it "handles multiple file changes with debouncing" do
    modified = false
    result, _streamed_out, _streamed_err = capture_watch_output(debounce: 200) do |out, err|
      # Wait for the watcher to be ready
      if !modified && err && err.include?("s/self/live@")
        # Rapidly modify multiple files
        calc_file = rux_ruby_dir.join("lib/calculator.rb")
        calc_file.write(calc_file.read + "\n# test")

        str_file = rux_ruby_dir.join("lib/string_utils.rb")
        str_file.write(str_file.read + "\n# test")

        val_file = rux_ruby_dir.join("lib/validator.rb")
        val_file.write(val_file.read + "\n# test")

        modified = true
      end
    end

    output = result.out + result.err

    # Should see all changes
    expect(output).to include("Changed: lib/calculator.rb")
    expect(output).to include("Changed: lib/string_utils.rb")
    expect(output).to include("Changed: lib/validator.rb")

    # Should run corresponding specs
    expect(output).to include("Running: spec/calculator_spec.rb")
    expect(output).to include("Running: spec/string_utils_spec.rb")
    expect(output).to include("Running: spec/validator_spec.rb")
  end

  describe "Rails-style mappings" do
    let(:app_dir) { File.join(rux_ruby_dir, "app") }

    before do
      # Create Rails-like directory structure
      FileUtils.mkdir_p(File.join(app_dir, "models"))
      FileUtils.mkdir_p(File.join(app_dir, "controllers"))
    end

    after do
      FileUtils.rm_rf(app_dir)
    end

    it "detects app directory and watches it" do
      result, _streamed_out, _streamed_err = capture_watch_output(rux_timeout: 2)

      output = result.out + result.err

      expect(output).to include("Watching directories: spec, lib, app")
    end

    it "maps app/models files to spec/models" do
      model_file = File.join(app_dir, "models/user.rb")
      File.write(model_file, "class User; end")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
        # Wait for the watcher to be ready
        if !modified && err && err.include?("s/self/live@")
          # Write to trigger a modify event
          File.write(model_file, "class User\n  # Modified\nend")
          modified = true
        end
      end

      output = result.out + result.err

      expect(output).to include("Changed: app/models/user.rb")
      expect(output).to include("Running: spec/models/user_spec.rb")
    end

    it "maps app/controllers files to spec/controllers" do
      controller_file = File.join(app_dir, "controllers/users_controller.rb")

      modified = false
      result, _streamed_out, _streamed_err = capture_watch_output do |out, err|
        # Wait for the watcher to be ready
        if !modified && err && err.include?("s/self/live@")
          # Write to trigger a modify event
          File.write(controller_file, "class UsersController\n  def index\n    # Modified\n  end\nend")
          modified = true
        end
      end

      output = result.out + result.err

      expect(output).to include("Changed: app/controllers/users_controller.rb")
      expect(output).to include("Running: spec/controllers/users_controller_spec.rb")
    end
  end
end
