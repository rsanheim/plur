require "spec_helper"
require "tmpdir"
require "fileutils"
require "json"

RSpec.describe "Rux runtime tracking" do
  let(:rux_binary) { File.join(__dir__, "..", "rux", "rux") }
  let(:test_project_path) { File.join(__dir__, "..", "rux-ruby") }

  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  describe "runtime data" do
    it "writes runtime data to specified directory with --runtime-dir flag" do
      Dir.mktmpdir do |tmpdir|
        runtime_dir = File.join(tmpdir, "runtime_data")

        Dir.chdir(test_project_path) do
          # Run rux with runtime tracking enabled
          `#{rux_binary} --runtime-dir #{runtime_dir} spec/calculator_spec.rb spec/string_utils_spec.rb 2>&1`

          expect($?.exitstatus).to eq(0)

          # Check that runtime file was created
          runtime_file = File.join(runtime_dir, "runtime.json")
          expect(File.exist?(runtime_file)).to be(true)

          # Verify minimal runtime data structure
          data = JSON.parse(File.read(runtime_file))

          # Simple format: just file paths mapped to runtimes
          expect(data).to have_key("./spec/calculator_spec.rb")
          expect(data).to have_key("./spec/string_utils_spec.rb")

          # Each file should have a positive runtime
          expect(data["./spec/calculator_spec.rb"]).to be > 0
          expect(data["./spec/string_utils_spec.rb"]).to be > 0
        end
      end
    end

    it "creates runtime directory if it doesn't exist" do
      Dir.mktmpdir do |tmpdir|
        runtime_dir = File.join(tmpdir, "nested", "runtime", "data")

        Dir.chdir(test_project_path) do
          `#{rux_binary} --runtime-dir #{runtime_dir} spec/calculator_spec.rb 2>&1`

          expect($?.exitstatus).to eq(0)
          expect(Dir.exist?(runtime_dir)).to be(true)

          runtime_file = File.join(runtime_dir, "runtime.json")
          expect(File.exist?(runtime_file)).to be(true)
        end
      end
    end

    it "accumulates runtime data across multiple workers" do
      Dir.mktmpdir do |tmpdir|
        runtime_dir = File.join(tmpdir, "runtime_data")

        Dir.chdir(test_project_path) do
          # Run with multiple workers
          `#{rux_binary} --runtime-dir #{runtime_dir} -n 4 2>&1`

          expect($?.exitstatus).to eq(0)

          # Runtime file should exist
          runtime_file = File.join(runtime_dir, "runtime.json")
          expect(File.exist?(runtime_file)).to be(true)

          data = JSON.parse(File.read(runtime_file))

          # Should have runtime data for all spec files
          # Just verify we got some files with runtimes
          expect(data.size).to be >= 9 # rux-ruby has 9 spec files

          # All values should be positive numbers
          data.each do |file_path, runtime|
            expect(runtime).to be_a(Numeric)
            expect(runtime).to be > 0
          end
        end
      end
    end

    it "records runtime data even when tests fail" do
      failing_specs_path = File.join(__dir__, "..", "test_fixtures", "failing_specs")

      Dir.mktmpdir do |tmpdir|
        runtime_dir = File.join(tmpdir, "runtime_data")

        Dir.chdir(failing_specs_path) do
          `#{rux_binary} --runtime-dir #{runtime_dir} spec/expectation_failures_spec.rb 2>&1`

          # Exit status should be 1 due to failures
          expect($?.exitstatus).to eq(1)

          # Runtime data should still be written
          runtime_file = File.join(runtime_dir, "runtime.json")
          expect(File.exist?(runtime_file)).to be(true)

          data = JSON.parse(File.read(runtime_file))
          expect(data).to have_key("./spec/expectation_failures_spec.rb")
          expect(data["./spec/expectation_failures_spec.rb"]).to be > 0
        end
      end
    end

    it "saves to ~/.cache/rux by default when no --runtime-dir is specified" do
      Dir.chdir(test_project_path) do
        # Run without --runtime-dir flag
        `#{rux_binary} spec/calculator_spec.rb 2>&1`

        expect($?.exitstatus).to eq(0)

        # Check default location following XDG Base Directory spec
        cache_dir = ENV["XDG_CACHE_HOME"] || File.join(ENV["HOME"], ".cache")
        default_runtime_file = File.join(cache_dir, "rux", "runtime.json")

        expect(File.exist?(default_runtime_file)).to be(true)

        # Verify the content
        data = JSON.parse(File.read(default_runtime_file))
        expect(data).to have_key("./spec/calculator_spec.rb")
        expect(data["./spec/calculator_spec.rb"]).to be > 0

        # Clean up
        FileUtils.rm_f(default_runtime_file)
      end
    end
  end
end
