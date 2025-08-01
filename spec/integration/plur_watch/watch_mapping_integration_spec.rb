require "spec_helper"

RSpec.describe "plur watch advanced file mapping" do
  include PlurWatchHelper
  # This spec focuses on testing with temporary directory structures
  # Basic file mapping is covered in watch_integration_spec.rb

  let(:temp_dir) { Dir.mktmpdir("plur-watch-mapping-test") }

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

  it "watches multiple directories including app when present" do
    result = run_plur_watch(dir: temp_dir, timeout: 1)
    expect(result.err).to include("directories=[spec lib app]")
  end

  it "runs the corresponding spec when a lib file changes" do
    result = run_plur_watch(dir: temp_dir, timeout: 3) do
      # Modify the file content to trigger a change event
      calc_file = "#{temp_dir}/lib/calculator.rb"
      content = File.read(calc_file)
      File.write(calc_file, content + "\n# modified")
    end

    expect_file_change_logged(result.err, "./lib/calculator.rb")
    expect_spec_run_logged(result.err, "./spec/calculator_spec.rb")
    expect(result.out).to include("1 example, 0 failures")
  end

  it "runs the corresponding model spec when a Rails model changes" do
    result = run_plur_watch(dir: temp_dir, timeout: 3) do
      # Modify the file content to trigger a change event
      user_file = "#{temp_dir}/app/models/user.rb"
      content = File.read(user_file)
      File.write(user_file, content + "\n# modified")
    end

    expect_file_change_logged(result.err, "./app/models/user.rb")
    expect_spec_run_logged(result.err, "./spec/models/user_spec.rb")
    expect(result.out).to include("1 example, 0 failures")
  end

  it "runs all specs when spec_helper.rb changes" do
    result = run_plur_watch(dir: temp_dir, timeout: 3) do
      # Modify the file content to trigger a change event
      spec_helper = "#{temp_dir}/spec/spec_helper.rb"
      content = File.read(spec_helper)
      File.write(spec_helper, content + "\n# modified")
    end

    expect_file_change_logged(result.err, "./spec/spec_helper.rb")
    expect_spec_run_logged(result.err, "./spec")
    # Should run all 3 specs
    expect(result.out).to include("3 examples, 0 failures")
  end

  it "handles nested lib files correctly" do
    result = run_plur_watch(dir: temp_dir, timeout: 3) do
      # Modify the file content to trigger a change event
      product_file = "#{temp_dir}/lib/models/product.rb"
      content = File.read(product_file)
      File.write(product_file, content + "\n# modified")
    end

    expect_file_change_logged(result.err, "./lib/models/product.rb")
    expect_spec_run_logged(result.err, "./spec/models/product_spec.rb")
    expect(result.out).to include("1 example, 0 failures")
  end
end
