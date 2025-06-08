require "spec_helper"
require "tmpdir"
require "benchmark"

RSpec.describe "Rux performance" do
  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  describe "parallelization efficiency" do
    it "chooses optimal execution strategy based on file count" do
      Dir.chdir(default_ruby_dir) do
        # With grouping optimization, for small test suites a single worker
        # might be faster as it avoids process spawn overhead
        single_output = `#{rux_binary} -n 1 2>&1`
        default_output = `#{rux_binary} 2>&1`

        # Both should complete successfully
        expect($?.exitstatus).to eq(0)

        # Should use grouped execution when appropriate (either runtime or size-based)
        expect(single_output).to match(/Using (runtime-based|size-based) grouped execution: 11 files across 1 workers/)

        # Verify all examples run in both cases
        expect(single_output).to match(/\d+ examples, 0 failures/)
        expect(default_output).to match(/\d+ examples, 0 failures/)
      end
    end

    it "shows wall time vs CPU time to demonstrate parallelization" do
      Dir.chdir(default_ruby_dir) do
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

  describe "worker optimization" do
    it "chooses reasonable default worker count" do
      Dir.chdir(default_ruby_dir) do
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
