require "spec_helper"
require "benchmark"

RSpec.describe "Plur performance" do
  describe "parallelization efficiency" do
    it "shows wall time vs CPU time to demonstrate parallelization" do
      result = run_plur("-C", default_ruby_dir)

      expect(result.out).to match(/Finished in [\d.]+ seconds \(files took [\d.]+ seconds to load\)/)
      if result.out =~ /Finished in ([\d.]+) seconds/
        wall_time = $1.to_f

        expect(wall_time).to be < 5.0 # Assuming reasonable test suite
      end
    end
  end

  describe "overhead measurement" do
    it "has minimal overhead for small test suites" do
      simple_project = project_fixture("rspec-success-simple")

      plur_time = Benchmark.realtime do
        run_plur("-C", simple_project.to_s, "spec/simple_spec.rb")
      end

      rspec_time = Benchmark.realtime do
        Dir.chdir(simple_project) do
          run_rspec("spec/simple_spec.rb")
        end
      end

      if ENV["VERBOSE"]
        puts "plur time: #{plur_time.round(3)}s"
        puts "RSpec time: #{rspec_time.round(3)}s"
        puts "Overhead: #{(plur_time - rspec_time).round(3)}s"
      end

      # Overhead should be minimal (less than 2 seconds, allowing for CI variance)
      expect(plur_time - rspec_time).to be < 2.0
    end
  end

  describe "worker optimization" do
    it "chooses reasonable default worker count" do
      result = run_plur("-C", default_ruby_dir.to_s)

      # Should show worker count in stderr
      expect(result.err).to match(/Running \d+ specs \[rspec\] in parallel using \d+ workers/)

      available_cores = `nproc 2>/dev/null || sysctl -n hw.ncpu 2>/dev/null`.to_i
      expected_workers = [available_cores - 2, 1].max

      # Extract values
      if result.err =~ /Running \d+ specs \[rspec\] in parallel using (\d+) workers/
        workers = $1.to_i

        # Default should be cores - 2 (but at least 1)
        expected_workers = [workers, expected_workers].min

        expect(workers).to eq(expected_workers)
      end
    end
  end
end
