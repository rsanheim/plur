require 'spec_helper'
require 'tempfile'
require 'fileutils'
require 'timeout'

RSpec.describe "rux watch integration" do
  let(:rux_ruby_dir) { File.join(File.dirname(__FILE__), '..', 'rux-ruby') }
  
  def capture_watch_output(timeout_seconds: 5, debounce: nil, &block)
    output = ""
    
    Dir.chdir(rux_ruby_dir) do
      # Start watch mode in a thread
      read_io, write_io = IO.pipe
      
      debounce_flag = debounce ? "--debounce #{debounce}" : ""
      watch_pid = spawn("rux watch --timeout #{timeout_seconds} #{debounce_flag}", 
                       out: write_io, err: write_io)
      write_io.close
      
      # Give watcher time to start
      sleep 1
      
      # Execute the block that will modify files
      yield if block_given?
      
      # Read output
      begin
        Timeout.timeout(timeout_seconds + 2) do
          output = read_io.read
        end
      rescue Timeout::Error
        # Expected - the watch command will timeout
      ensure
        read_io.close
        Process.wait(watch_pid) rescue nil
      end
    end
    
    output
  end
  
  it "starts watching the correct directories" do
    output = capture_watch_output(timeout_seconds: 2)
    
    expect(output).to include("Starting rux watch mode")
    expect(output).to include("Watching directories: spec, lib")
    expect(output).to include("Debounce delay: 100ms")
  end
  
  it "maps lib files to their specs" do
    output = capture_watch_output do
      # Touch a lib file
      FileUtils.touch(File.join(rux_ruby_dir, "lib/calculator.rb"))
      sleep 0.5
    end
    
    expect(output).to include("Changed: lib/calculator.rb")
    expect(output).to include("Running: spec/calculator_spec.rb")
  end
  
  it "maps nested lib files correctly" do
    # Create a nested lib file temporarily
    nested_lib = File.join(rux_ruby_dir, "lib/models/temp_model.rb")
    FileUtils.mkdir_p(File.dirname(nested_lib))
    
    begin
      File.write(nested_lib, "class TempModel; end")
      
      output = capture_watch_output do
        FileUtils.touch(nested_lib)
        sleep 0.5
      end
      
      expect(output).to include("Changed: lib/models/temp_model.rb")
      expect(output).to include("Running: spec/models/temp_model_spec.rb")
      # It will say spec not found since we didn't create the spec
      expect(output).to include("Spec file not found: spec/models/temp_model_spec.rb")
    ensure
      FileUtils.rm_f(nested_lib)
      FileUtils.rmdir(File.dirname(nested_lib)) rescue nil
    end
  end
  
  it "runs all specs when spec_helper.rb changes" do
    output = capture_watch_output do
      FileUtils.touch(File.join(rux_ruby_dir, "spec/spec_helper.rb"))
      sleep 0.5
    end
    
    expect(output).to include("Changed: spec/spec_helper.rb")
    expect(output).to include("Running: spec")
    expect(output).to include("Running all specs in spec/")
  end
  
  it "handles spec file changes" do
    output = capture_watch_output do
      FileUtils.touch(File.join(rux_ruby_dir, "spec/calculator_spec.rb"))
      sleep 0.5
    end
    
    expect(output).to include("Changed: spec/calculator_spec.rb")
    expect(output).to include("Running: spec/calculator_spec.rb")
  end
  
  it "ignores non-Ruby files" do
    readme_file = File.join(rux_ruby_dir, "README.md")
    FileUtils.touch(readme_file) # Ensure it exists
    
    output = capture_watch_output do
      FileUtils.touch(readme_file)
      sleep 0.5
    end
    
    # Should not see any change events for README.md
    expect(output).not_to include("Changed: README.md")
  end
  
  it "respects custom debounce delay" do
    output = capture_watch_output(timeout_seconds: 2, debounce: 250)
    
    expect(output).to include("Debounce delay: 250ms")
  end
  
  it "handles multiple file changes with debouncing" do
    output = capture_watch_output(debounce: 200) do
      # Rapidly touch multiple files
      FileUtils.touch(File.join(rux_ruby_dir, "lib/calculator.rb"))
      FileUtils.touch(File.join(rux_ruby_dir, "lib/string_utils.rb"))
      FileUtils.touch(File.join(rux_ruby_dir, "lib/validator.rb"))
      sleep 0.5
    end
    
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
      output = capture_watch_output(timeout_seconds: 2)
      
      expect(output).to include("Watching directories: spec, lib, app")
    end
    
    it "maps app/models files to spec/models" do
      model_file = File.join(app_dir, "models/user.rb")
      File.write(model_file, "class User; end")
      
      output = capture_watch_output do
        FileUtils.touch(model_file)
        sleep 0.5
      end
      
      expect(output).to include("Changed: app/models/user.rb")
      expect(output).to include("Running: spec/models/user_spec.rb")
    end
    
    it "maps app/controllers files to spec/controllers" do
      controller_file = File.join(app_dir, "controllers/users_controller.rb")
      File.write(controller_file, "class UsersController; end")
      
      output = capture_watch_output do
        FileUtils.touch(controller_file)
        sleep 0.5
      end
      
      expect(output).to include("Changed: app/controllers/users_controller.rb")
      expect(output).to include("Running: spec/controllers/users_controller_spec.rb")
    end
  end
end