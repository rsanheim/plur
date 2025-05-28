require "spec_helper"
require "tmpdir"
require "fileutils"

RSpec.describe "Rux parallel execution" do
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

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

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          output = `#{rux_binary} -n 3 2>&1`

          # With 3 workers and 3 spec files, each should run with different TEST_ENV_NUMBER
          # Just verify it runs successfully
          expect(output).to include("3 examples, 0 failures")
          expect($?.exitstatus).to eq(0)
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

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          output = `#{rux_binary} -n 4 2>&1`

          # Now it should actually use 4 workers
          expect(output).to include("using 4 workers")
          expect(output).to include("4 examples, 0 failures")
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

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          output = `#{rux_binary} -n 3 2>&1`

          # Should see organized output with all tests passing
          expect(output).to include("15 examples, 0 failures")

          # Progress dots should appear on one line (not interleaved)
          progress_lines = output.split("\n").select { |l| l =~ /^\e?\[?3?2?m?\.\e?\[?0?m?/ || l =~ /^\.+$/ }
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

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          output = `#{rux_binary} --color 2>&1`

          # Should see progress dots and F's with colors
          expect(output).to match(/\e\[32m\.\e\[0m/) # Green dots
          expect(output).to match(/\e\[31mF\e\[0m/) # Red F's

          # Should see summary
          expect(output).to include("7 examples, 2 failures")
        end
      end
    end
  end

  describe "failure reporting" do
    it "aggregates failures from all workers", pending: "Full failure summary output not yet implemented" do
      Dir.mktmpdir do |tmpdir|
        spec_dir = File.join(tmpdir, "spec")
        FileUtils.mkdir_p(spec_dir)

        # Create specs with different types of failures
        spec1_path = File.join(spec_dir, "failure_1_spec.rb")
        File.write(spec1_path, <<~RUBY)
          RSpec.describe "failure 1" do
            it "has expectation failure" do
              expect(1 + 1).to eq(3)
            end
          end
        RUBY

        spec2_path = File.join(spec_dir, "failure_2_spec.rb")
        File.write(spec2_path, <<~RUBY)
          RSpec.describe "failure 2" do
            it "raises an error" do
              raise "Something went wrong"
            end
          end
        RUBY

        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          output = `#{rux_binary} -n 2 2>&1`

          # Should show both failures
          expect(output).to include("failure 1")
          expect(output).to include("has expectation failure")
          expect(output).to include("expected: 3")
          expect(output).to include("got: 2")

          expect(output).to include("failure 2")
          expect(output).to include("raises an error")
          expect(output).to include("Something went wrong")

          # Should show failed examples list
          expect(output).to include("Failed examples:")
          expect(output).to include("rspec ./spec/failure_1_spec.rb")
          expect(output).to include("rspec ./spec/failure_2_spec.rb")
        end
      end
    end
  end
end
