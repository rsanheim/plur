require "spec_helper"
require "tempfile"
require "fileutils"

RSpec.describe "Framework Selection" do
  let(:test_dir) { Dir.mktmpdir("plur-framework-test") }

  after do
    FileUtils.rm_rf(test_dir) if test_dir && Dir.exist?(test_dir)
  end

  describe "with both spec/ and test/ directories" do
    before do
      FileUtils.mkdir_p(File.join(test_dir, "spec"))
      FileUtils.mkdir_p(File.join(test_dir, "test"))

      # Create a passing RSpec test
      File.write(File.join(test_dir, "spec", "example_spec.rb"), <<~RUBY)
        RSpec.describe "Example" do
          it "passes" do
            expect(1).to eq(1)
          end
        end
      RUBY

      # Create a passing Minitest test
      File.write(File.join(test_dir, "test", "example_test.rb"), <<~RUBY)
        require 'minitest/autorun'

        class ExampleTest < Minitest::Test
          def test_passes
            assert_equal 1, 1
          end
        end
      RUBY

      # Create minimal Gemfile for both frameworks
      File.write(File.join(test_dir, "Gemfile"), <<~RUBY)
        source 'https://rubygems.org'
        gem 'rspec', '~> 3.0'
        gem 'minitest', '~> 5.0'
      RUBY

      # Run bundle install
      Dir.chdir(test_dir) do
        system("bundle install --quiet", out: File::NULL, err: File::NULL)
      end
    end

    it "defaults to RSpec when no --use flag is provided" do
      output = run_plur_in_dir(test_dir, "--dry-run")

      expect(output).to include("spec/example_spec.rb")
      expect(output).not_to include("test/example_test.rb")
    end

    it "runs RSpec tests when -t rspec is specified" do
      output = run_plur_in_dir(test_dir, "spec -t rspec --dry-run")

      expect(output).to include("spec/example_spec.rb")
      expect(output).not_to include("test/example_test.rb")
    end

    it "runs Minitest tests when -t minitest is specified" do
      output = run_plur_in_dir(test_dir, "spec -t minitest --dry-run")

      expect(output).to include("test/example_test.rb")
      expect(output).not_to include("spec/example_spec.rb")
    end

    context "with config file setting use=minitest" do
      before do
        File.write(File.join(test_dir, ".plur.toml"), <<~TOML)
          use = "minitest"
        TOML
      end

      it "uses Minitest as default" do
        output = run_plur_in_dir(test_dir, "--dry-run")

        expect(output).to include("test/example_test.rb")
        expect(output).not_to include("spec/example_spec.rb")
      end

      it "allows CLI flag to override config file" do
        output = run_plur_in_dir(test_dir, "spec -t rspec --dry-run")

        expect(output).to include("spec/example_spec.rb")
        expect(output).not_to include("test/example_test.rb")
      end
    end
  end

  describe "with only spec/ directory" do
    before do
      FileUtils.mkdir_p(File.join(test_dir, "spec"))
      File.write(File.join(test_dir, "spec", "example_spec.rb"), <<~RUBY)
        RSpec.describe "Example" do
          it "passes" do
            expect(1).to eq(1)
          end
        end
      RUBY
    end

    it "detects and runs RSpec tests" do
      output = run_plur_in_dir(test_dir, "--dry-run")

      expect(output).to include("spec/example_spec.rb")
    end
  end

  describe "with only test/ directory" do
    before do
      FileUtils.mkdir_p(File.join(test_dir, "test"))
      File.write(File.join(test_dir, "test", "example_test.rb"), <<~RUBY)
        require 'minitest/autorun'

        class ExampleTest < Minitest::Test
          def test_passes
            assert_equal 1, 1
          end
        end
      RUBY
    end

    it "detects and runs Minitest tests" do
      output = run_plur_in_dir(test_dir, "--dry-run")

      expect(output).to include("test/example_test.rb")
    end
  end

  describe "with neither spec/ nor test/ directory" do
    it "defaults to RSpec for backward compatibility" do
      # This will show no files to run, but we can check the detected framework
      # by looking at the error message
      output = run_plur_in_dir(test_dir, "--dry-run", allow_failure: true)

      # The error message should mention looking for RSpec files
      expect(output).to include("*_spec.rb")
      expect(output).to include("spec/")
      # And not mention Minitest patterns
      expect(output).not_to include("*_test.rb")
      expect(output).not_to include("test/")
    end
  end

  private

  def run_plur_in_dir(dir, args = "", allow_failure: false)
    Dir.chdir(dir) do
      output = `plur #{args} 2>&1`
      unless $?.success?
        # For dry-run, it's ok if plur exits with non-zero when no tests found
        if allow_failure || (args.include?("--dry-run") && output.include?("no test files found"))
          return output
        end
        raise "plur failed: #{output}"
      end
      output
    end
  end
end
