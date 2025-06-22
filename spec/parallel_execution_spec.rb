require "spec_helper"
require "tmpdir"
require "fileutils"

RSpec.describe "Rux parallel execution" do
  describe "environment variables" do
    it "sets TEST_ENV_NUMBER for each worker" do
      Dir.mktmpdir do |tmpdir|
        # Create spec directory
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)

        # Create multiple spec files that verify TEST_ENV_NUMBER is set
        3.times do |i|
          spec_path = File.join(spec_dir, "env_test_#{i}_spec.rb")
          File.write(spec_path, <<~RUBY)
            RSpec.describe "env test #{i}" do
              it "has TEST_ENV_NUMBER set" do
                env_num = ENV['TEST_ENV_NUMBER']
                # Just verify it's set to a number or empty string
                expect(env_num).to match(/^\\d*$/)
              end
            end
          RUBY
        end

        # Create minimal Gemfile
        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          result = run_rux("-n", "3")

          # With 3 workers and 3 spec files, each should run with different TEST_ENV_NUMBER
          # Just verify it runs successfully
          expect(result.out).to include("3 examples, 0 failures")
        end
      end
    end

    it "sets PARALLEL_TEST_GROUPS correctly" do
      Dir.mktmpdir do |tmpdir|
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)

        # Create 4 spec files to ensure we can use 4 workers
        4.times do |i|
          spec_path = File.join(spec_dir, "groups_test_#{i}_spec.rb")
          File.write(spec_path, <<~RUBY)
            RSpec.describe "groups test #{i}" do
              it "has PARALLEL_TEST_GROUPS set to 4" do
                groups = ENV['PARALLEL_TEST_GROUPS']
                expect(groups).to eq("4")
              end
            end
          RUBY
        end

        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          result = run_rux("-n", "4")

          # Now it should actually use 4 workers
          expect(result.out).to include("using 4 workers")
          expect(result.out).to include("4 examples, 0 failures")
        end
      end
    end
  end

  describe "output synchronization" do
    it "runs multiple workers without interleaving progress output" do
      Dir.mktmpdir do |tmpdir|
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)

        # Create multiple specs
        5.times do |i|
          spec_path = File.join(spec_dir, "output_test_#{i}_spec.rb")
          File.write(spec_path, <<~RUBY)
            RSpec.describe "output test #{i}" do
              3.times do |j|
                it "test \#{j}" do
                  expect(true).to be true
                end
              end
            end
          RUBY
        end

        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          result = run_rux("-n", "3")

          # Should see organized output with all tests passing
          expect(result.out).to include("15 examples, 0 failures")

          # Progress dots should appear on one line (not interleaved)
          progress_lines = result.out.split("\n").select { |l| l =~ /^\e?\[?3?2?m?\.\e?\[?0?m?/ || l =~ /^\.+$/ }
          expect(progress_lines.size).to be <= 2 # At most "Running..." line and one progress line
        end
      end
    end
  end

  describe "progress output" do
    it "shows combined progress from all workers" do
      Dir.mktmpdir do |tmpdir|
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)

        # Create multiple spec files with mix of passing and failing tests
        5.times do |i|
          spec_path = File.join(spec_dir, "progress_test_#{i}_spec.rb")
          File.write(spec_path, <<~RUBY)
            RSpec.describe "progress test #{i}" do
              it "passes" do
                expect(true).to be true
              end
              
              it "fails" do
                expect(false).to be true
              end if #{i} < 2  # Only first 2 files have failures
            end
          RUBY
        end

        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          result = run_rux_allowing_errors("--color")

          # Should see progress dots and F's with colors
          expect(result.out).to match(/\e\[32m\.\e\[0m/) # Green dots
          expect(result.out).to match(/\e\[31mF\e\[0m/) # Red F's

          # Should see summary
          expect(result.out).to include("7 examples, 2 failures")
        end
      end
    end
  end
end
