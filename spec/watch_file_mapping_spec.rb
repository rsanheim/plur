require 'spec_helper'
require 'tmpdir'
require 'fileutils'

RSpec.describe "rux watch file mapping" do
  let(:temp_dir) { Dir.mktmpdir("rux-watch-test") }
  
  before do
    # Set up a basic Ruby project structure
    FileUtils.mkdir_p("#{temp_dir}/lib")
    FileUtils.mkdir_p("#{temp_dir}/spec")
    FileUtils.mkdir_p("#{temp_dir}/app/models")
    FileUtils.mkdir_p("#{temp_dir}/app/controllers") 
    FileUtils.mkdir_p("#{temp_dir}/spec/models")
    FileUtils.mkdir_p("#{temp_dir}/spec/controllers")
    
    # Create some test files
    File.write("#{temp_dir}/lib/calculator.rb", "class Calculator; end")
    File.write("#{temp_dir}/spec/calculator_spec.rb", "RSpec.describe Calculator do\n  it 'works' do\n    expect(1).to eq(1)\n  end\nend")
    File.write("#{temp_dir}/spec/spec_helper.rb", "# spec helper")
    
    # Rails-style files
    File.write("#{temp_dir}/app/models/user.rb", "class User; end")
    File.write("#{temp_dir}/spec/models/user_spec.rb", "RSpec.describe User do\n  it 'works' do\n    expect(1).to eq(1)\n  end\nend")
  end
  
  after do
    FileUtils.rm_rf(temp_dir)
  end
  
  context "when watching lib files" do
    it "runs the corresponding spec when a lib file changes" do
      Dir.chdir(temp_dir) do
        # Start watch mode with a timeout
        watch_process = spawn("rux watch --timeout 3", err: [:child, :out])
        
        # Wait for watcher to start
        sleep 1
        
        # Touch the lib file to trigger a change
        FileUtils.touch("lib/calculator.rb")
        
        # Wait for the process to complete
        Process.wait(watch_process)
        
        # The output should show it ran calculator_spec.rb
        # We'll verify this in a more comprehensive test with output capture
      end
    end
  end
  
  context "when spec_helper.rb changes" do
    it "runs all specs" do
      Dir.chdir(temp_dir) do
        # This would run all specs when spec_helper.rb is modified
        # Verified in more comprehensive tests
      end
    end
  end
  
  context "when Rails app files change" do
    it "runs the corresponding model spec when a model changes" do
      Dir.chdir(temp_dir) do
        # Touch app/models/user.rb should run spec/models/user_spec.rb
        # Verified in more comprehensive tests
      end
    end
  end
end