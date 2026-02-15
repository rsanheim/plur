require "spec_helper"
require "tempfile"
require "fileutils"

RSpec.describe "Framework Selection" do
  let(:test_dir) { Dir.mktmpdir("plur-framework-test") }

  after do
    FileUtils.rm_rf(test_dir) if test_dir && Dir.exist?(test_dir)
  end

  describe "with both spec/ and test/ directories" do
    let(:project_dir) { project_fixture!("mixed-rspec-minitest") }

    before(:all) do
      # Ensure dependencies are installed
      chdir(project_fixture!("mixed-rspec-minitest")) do
        Bundler.with_unbundled_env do
          system("bundle check", out: File::NULL, err: File::NULL) ||
            system("bundle install", out: File::NULL, err: File::NULL, exception: true)
        end
      end
    end

    it "defaults to RSpec when no --use flag is provided" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("--dry-run")
          expect(result).to be_success
          expect(result.err).to include("spec/example_spec.rb")
          expect(result.err).not_to include("test/example_test.rb")
        end
      end
    end

    it "runs RSpec tests when -t rspec is specified" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("spec", "-u", "rspec", "--dry-run")
          expect(result).to be_success
          expect(result.err).to include("spec/example_spec.rb")
          expect(result.err).not_to include("test/example_test.rb")
        end
      end
    end

    it "runs Minitest tests when -t minitest is specified" do
      chdir(project_dir) do
        Bundler.with_unbundled_env do
          result = run_plur("spec", "-u", "minitest", "--dry-run")
          expect(result).to be_success
          expect(result.err).to include("test/example_test.rb")
          expect(result.err).not_to include("spec/example_spec.rb")
        end
      end
    end

    context "with config file setting use=minitest" do
      let(:config_file) { project_dir.join(".plur.toml") }

      before do
        File.write(config_file, "use = \"minitest\"\n")
      end

      after do
        File.delete(config_file) if File.exist?(config_file)
      end

      it "uses Minitest as default" do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            result = run_plur("--dry-run")
            expect(result).to be_success
            expect(result.err).to include("test/example_test.rb")
            expect(result.err).not_to include("spec/example_spec.rb")
          end
        end
      end

      it "allows CLI flag to override config file" do
        chdir(project_dir) do
          Bundler.with_unbundled_env do
            result = run_plur("spec", "-u", "rspec", "--dry-run")
            expect(result).to be_success
            expect(result.err).to include("spec/example_spec.rb")
            expect(result.err).not_to include("test/example_test.rb")
          end
        end
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
    it "fails with clear error message when no framework detected" do
      # With no indicators (Gemfile, spec/, test/, lib/), should fail with clear message
      output = run_plur_in_dir(test_dir, "--dry-run", allow_failure: true)

      # The error message should indicate no framework was detected
      expect(output).to include("no default spec/test files found")
    end
  end

  private

  def run_plur_in_dir(dir, args = "", allow_failure: false)
    Dir.chdir(dir) do
      output = `#{plur_binary} #{args} 2>&1`
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
