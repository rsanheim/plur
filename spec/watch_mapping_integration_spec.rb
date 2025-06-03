require 'spec_helper'
require 'tempfile'
require 'fileutils'

RSpec.describe "rux watch file mapping integration" do
  let(:temp_dir) { Dir.mktmpdir("rux-watch-mapping-test") }
  
  before do
    # Create a basic Ruby project structure
    FileUtils.mkdir_p("#{temp_dir}/lib/models")
    FileUtils.mkdir_p("#{temp_dir}/spec/models")
    FileUtils.mkdir_p("#{temp_dir}/app/models")
    FileUtils.mkdir_p("#{temp_dir}/app/controllers")
    FileUtils.mkdir_p("#{temp_dir}/spec/controllers")
    
    # Create test files with simple specs that pass
    File.write("#{temp_dir}/lib/calculator.rb", <<~RUBY)
      class Calculator
        def add(a, b)
          a + b
        end
      end
    RUBY
    
    File.write("#{temp_dir}/spec/calculator_spec.rb", <<~RUBY)
      require_relative '../lib/calculator'
      
      RSpec.describe Calculator do
        it 'adds numbers' do
          expect(Calculator.new.add(1, 2)).to eq(3)
        end
      end
    RUBY
    
    File.write("#{temp_dir}/spec/spec_helper.rb", <<~RUBY)
      RSpec.configure do |config|
        config.formatter = :progress
      end
    RUBY
    
    # Rails-style files
    File.write("#{temp_dir}/app/models/user.rb", <<~RUBY)
      class User
        attr_reader :name
        
        def initialize(name)
          @name = name
        end
      end
    RUBY
    
    File.write("#{temp_dir}/spec/models/user_spec.rb", <<~RUBY)
      require_relative '../../app/models/user'
      
      RSpec.describe User do
        it 'has a name' do
          user = User.new('Alice')
          expect(user.name).to eq('Alice')
        end
      end
    RUBY
    
    # Nested lib file
    File.write("#{temp_dir}/lib/models/product.rb", <<~RUBY)
      class Product
        attr_reader :title
        
        def initialize(title)
          @title = title
        end
      end
    RUBY
    
    File.write("#{temp_dir}/spec/models/product_spec.rb", <<~RUBY)
      require_relative '../../lib/models/product'
      
      RSpec.describe Product do
        it 'has a title' do
          product = Product.new('Widget')
          expect(product.title).to eq('Widget')
        end
      end
    RUBY
  end
  
  after do
    FileUtils.rm_rf(temp_dir)
  end
  
  def simulate_file_change_and_capture_output(file_to_change, expected_spec)
    Dir.chdir(temp_dir) do
      output = nil
      
      # Start watch mode in a thread
      watch_thread = Thread.new do
        output = `timeout 3 rux watch 2>&1`
      end
      
      # Wait for watcher to start
      sleep 1
      
      # Touch the file to trigger a change
      FileUtils.touch(file_to_change)
      
      # Wait for the thread to complete
      watch_thread.join
      
      output
    end
  end
  
  it "runs the corresponding spec when a lib file changes" do
    output = simulate_file_change_and_capture_output("lib/calculator.rb", "spec/calculator_spec.rb")
    
    expect(output).to include("Changed: lib/calculator.rb")
    expect(output).to include("Running: spec/calculator_spec.rb")
    expect(output).to include("1 example, 0 failures")
  end
  
  it "runs the corresponding model spec when a Rails model changes" do
    output = simulate_file_change_and_capture_output("app/models/user.rb", "spec/models/user_spec.rb")
    
    expect(output).to include("Changed: app/models/user.rb")
    expect(output).to include("Running: spec/models/user_spec.rb")
    expect(output).to include("1 example, 0 failures")
  end
  
  it "runs all specs when spec_helper.rb changes" do
    output = simulate_file_change_and_capture_output("spec/spec_helper.rb", "all specs")
    
    expect(output).to include("Changed: spec/spec_helper.rb")
    expect(output).to include("Running: spec")
    expect(output).to include("Running all specs in spec/")
    # Should run all 3 specs
    expect(output).to include("3 examples, 0 failures")
  end
  
  it "handles nested lib files correctly" do
    output = simulate_file_change_and_capture_output("lib/models/product.rb", "spec/models/product_spec.rb")
    
    expect(output).to include("Changed: lib/models/product.rb")
    expect(output).to include("Running: spec/models/product_spec.rb")
    expect(output).to include("1 example, 0 failures")
  end
  
  it "watches multiple directories" do
    Dir.chdir(temp_dir) do
      output = `timeout 1 rux watch 2>&1`
      expect(output).to include("Watching directories: spec, lib, app")
    end
  end
end