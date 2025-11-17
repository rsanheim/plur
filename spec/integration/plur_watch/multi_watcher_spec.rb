require "spec_helper"
require "tempfile"
require "fileutils"

RSpec.describe "plur watch with multiple directories" do
  include PlurWatchHelper

  it "watches multiple directories simultaneously" do
    Dir.mktmpdir do |tmpdir|
      # Create test structure with multiple directories
      spec_dir = File.join(tmpdir, "spec")
      lib_dir = File.join(tmpdir, "lib")
      app_dir = File.join(tmpdir, "app")

      FileUtils.mkdir_p([spec_dir, lib_dir, app_dir])

      # Create initial files
      spec_file = File.join(spec_dir, "example_spec.rb")
      File.write(spec_file, "RSpec.describe 'Example' do\n  it 'works' do\n    expect(1).to eq(1)\n  end\nend")

      lib_file = File.join(lib_dir, "example.rb")
      File.write(lib_file, "class Example\nend")

      # Run watch with a short timeout
      result = run_plur_watch(dir: tmpdir, timeout: 3) do
        # Modify files in both directories immediately
        File.write(spec_file, "# Modified\n" + File.read(spec_file))
        File.write(lib_file, "# Modified\n" + File.read(lib_file))
      end

      # Verify we see watcher processes for both directories in stderr
      expect(result.err).to include("watch event=create type=watcher")
      expect(result.err).to include("/spec")
      expect(result.err).to include("/lib")

      # Verify both watchers went live (the output includes the full path)
      # The live messages are in stderr
      expect(result.err).to include("s/self/live@")
      expect(result.err).to include(spec_dir)
      expect(result.err).to include(lib_dir)
    end
  end

  it "starts watchers for existing directories only" do
    Dir.mktmpdir do |tmpdir|
      # Create only spec and lib directories, but not app
      spec_dir = File.join(tmpdir, "spec")
      lib_dir = File.join(tmpdir, "lib")
      FileUtils.mkdir_p([spec_dir, lib_dir])

      result = run_plur_watch(dir: tmpdir, timeout: 1)

      # Should start watchers for spec and lib only (in stderr)
      expect(result.err).to include("directories=[lib spec]")
      expect(result.err).not_to include("directories=[app lib spec]")
    end
  end

  it "cleanly shuts down all watchers on exit" do
    Dir.mktmpdir do |tmpdir|
      spec_dir = File.join(tmpdir, "spec")
      lib_dir = File.join(tmpdir, "lib")
      FileUtils.mkdir_p([spec_dir, lib_dir])

      result = run_plur_watch(dir: tmpdir, timeout: 1)

      # Verify clean shutdown messages
      expect(result.out).to include("Timeout reached, exiting!")

      # The debug output goes to stderr but that's expected behavior
      # Just verify no actual errors occurred
      expect(result.success?).to be true
    end
  end
end
