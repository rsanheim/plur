require "spec_helper"
require "tmpdir"
require "benchmark"

RSpec.describe "Rux performance" do
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }
  let(:test_project_path) { File.join(__dir__, "..", "rux-ruby") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  describe "parallelization efficiency" do
    it "runs faster with multiple workers than single worker" do
      Dir.chdir(test_project_path) do
        # Run with single worker
        single_time = Benchmark.realtime do
          system("#{rux_binary} -n 1", out: File::NULL, err: File::NULL)
        end

        # Run with multiple workers (default)
        parallel_time = Benchmark.realtime do
          system(rux_binary, out: File::NULL, err: File::NULL)
        end

        # Parallel should be faster (allowing some margin for overhead)
        expect(parallel_time).to be < (single_time * 0.9)
      end
    end

    it "shows wall time vs CPU time to demonstrate parallelization" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} 2>&1`

        # Should show both wall time and CPU time
        expect(output).to match(/Finished in [\d.]+ seconds \(files took [\d.]+ seconds to load\)/)

        # Extract times from output
        if output =~ /Finished in ([\d.]+) seconds/
          wall_time = $1.to_f

          # Wall time should be reasonable
          expect(wall_time).to be < 5.0 # Assuming reasonable test suite
        end
      end
    end
  end

  describe "overhead measurement" do
    it "has minimal overhead for small test suites" do
      Dir.mktmpdir do |tmpdir|
        # Create a single simple spec
        spec_path = File.join(tmpdir, "simple_spec.rb")
        File.write(spec_path, <<~RUBY)
          RSpec.describe "simple" do
            it "passes quickly" do
              expect(true).to be true
            end
          end
        RUBY

        File.write(File.join(tmpdir, "Gemfile"), <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)

          # Measure rux time
          rux_time = Benchmark.realtime do
            system("#{rux_binary} simple_spec.rb", out: File::NULL, err: File::NULL)
          end

          # Measure direct rspec time
          rspec_time = Benchmark.realtime do
            system("bundle exec rspec simple_spec.rb", out: File::NULL, err: File::NULL)
          end

          # Overhead should be minimal (less than 1 second)
          overhead = rux_time - rspec_time
          expect(overhead).to be < 1.0
        end
      end
    end
  end

  describe "scalability" do
    it "efficiently handles many spec files", pending: "Performance test is flaky due to timing variations" do
      Dir.mktmpdir do |tmpdir|
        # Create many small spec files
        20.times do |i|
          spec_path = File.join(tmpdir, "spec_#{i}_spec.rb")
          File.write(spec_path, <<~RUBY)
            RSpec.describe "spec #{i}" do
              it "test 1" do
                expect(true).to be true
              end
              
              it "test 2" do
                expect(1 + 1).to eq(2)
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

          # Time with different worker counts
          times = {}

          [1, 2, 4, 8].each do |workers|
            times[workers] = Benchmark.realtime do
              system("#{rux_binary} -n #{workers}", out: File::NULL, err: File::NULL)
            end
          end

          # Should see improvement with more workers (up to a point)
          expect(times[2]).to be < times[1]
          expect(times[4]).to be < times[2]

          # But diminishing returns at some point
          # (8 workers might not be much faster than 4 for only 20 files)
        end
      end
    end
  end

  describe "worker optimization" do
    it "chooses reasonable default worker count" do
      Dir.chdir(test_project_path) do
        output = `#{rux_binary} 2>&1`

        # Should show worker count and available cores
        expect(output).to match(/using \d+ workers \(\d+ cores available\)/)

        # Extract values
        if output =~ /using (\d+) workers \((\d+) cores available\)/
          workers = $1.to_i
          cores = $2.to_i

          # Default should be cores - 2 (but at least 1)
          expected_workers = [cores - 2, 1].max

          # But also limited by number of spec files
          spec_files = 11 # Known number in rux-ruby
          expected_workers = [expected_workers, spec_files].min

          expect(workers).to eq(expected_workers)
        end
      end
    end
  end
end
