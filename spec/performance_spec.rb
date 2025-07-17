require "spec_helper"
require "benchmark"

RSpec.describe "Plur performance" do
  let(:expected_spec_files) { 12 }

  describe "parallelization efficiency" do
    it "chooses optimal execution strategy based on file count" do
      # With grouping optimization, for small test suites a single worker
      # might be faster as it avoids process spawn overhead
      single_result = run_plur("-C", default_ruby_dir.to_s, "-n", "1")
      default_result = run_plur("-C", default_ruby_dir.to_s)

      # Should use grouped execution when appropriate (either runtime or size-based)
      expect(single_result.err).to match(/Using (runtime-based|size-based) grouped execution: #{expected_spec_files} files across 1 workers/)

      # Verify all examples run in both cases
      expect(single_result.out).to match(/\d+ examples, 0 failures/)
      expect(default_result.out).to match(/\d+ examples, 0 failures/)
    end

    it "shows wall time vs CPU time to demonstrate parallelization" do
      result = run_plur("-C", default_ruby_dir)

      # Should show both wall time and CPU time
      expect(result.out).to match(/Finished in [\d.]+ seconds \(files took [\d.]+ seconds to load\)/)

      # Extract times from output
      if result.out =~ /Finished in ([\d.]+) seconds/
        wall_time = $1.to_f

        # Wall time should be reasonable
        expect(wall_time).to be < 5.0 # Assuming reasonable test suite
      end
    end
  end

  describe "overhead measurement" do
    it "has minimal overhead for small test suites" do
      simple_project = project_fixture("rspec-success-simple")

      # Measure plur time
      plur_time = Benchmark.realtime do
        system("plur", "-C", simple_project.to_s, "spec/simple_spec.rb")
      end

      # Measure direct rspec time
      rspec_time = Benchmark.realtime do
        Dir.chdir(simple_project) do
          system("bundle exec rspec spec/simple_spec.rb")
        end
      end

      if ENV["VERBOSE"]
        # Output timing information for verification
        puts "plur time: #{plur_time.round(3)}s"
        puts "RSpec time: #{rspec_time.round(3)}s"
        puts "Overhead: #{(plur_time - rspec_time).round(3)}s"
      end

      # Overhead should be minimal (less than 1 second)
      expect(plur_time - rspec_time).to be < 1.0
    end
  end

  describe "worker optimization" do
    it "chooses reasonable default worker count" do
      result = run_plur("-C", default_ruby_dir.to_s)

      # Should show worker count and available cores
      expect(result.out).to match(/using \d+ workers \(\d+ cores available\)/)

      # Extract values
      if result.out =~ /using (\d+) workers \((\d+) cores available\)/
        workers = $1.to_i
        cores = $2.to_i

        # Default should be cores - 2 (but at least 1)
        expected_workers = [cores - 2, 1].max

        # But also limited by number of spec files
        spec_files = expected_spec_files
        expected_workers = [expected_workers, spec_files].min

        expect(workers).to eq(expected_workers)
      end
    end
  end
end
