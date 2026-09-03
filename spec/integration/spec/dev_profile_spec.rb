require "spec_helper"

RSpec.describe "--dev-profile" do
  def plur_binary
    File.expand_path("../../../plur", __dir__)
  end

  it "writes CPU, heap, goroutine and goroutine-leak profiles for the run" do
    Dir.mktmpdir do |dir|
      result = run_plur("-C", default_ruby_dir, "-n", "2", "--dev-profile", dir)

      expect(result.err).to include("[dev-profile] wrote profiles to")
      process_dirs = Dir.children(dir)
      expect(process_dirs.size).to eq(1)
      expect(process_dirs.first).to match(/\Aplur-\d+\z/)

      profile_dir = File.join(dir, process_dirs.first)
      expect(Dir.children(profile_dir).sort).to eq(%w[cpu.pprof goroutine.txt goroutineleak.txt heap.pprof])
      expect(File.read(File.join(profile_dir, "goroutineleak.txt"))).to start_with("goroutineleak profile: total 0")
    end
  end

  it "still writes profiles when the test run fails" do
    Dir.mktmpdir do |dir|
      result = run_plur("-C", project_fixture!("failing_specs"), "-n", "2", "--dev-profile", dir, allow_error: true)

      expect(result.exit_status).to eq(1)
      expect(Dir.glob(File.join(dir, "plur-*", "goroutineleak.txt")).size).to eq(1)
    end
  end
end
