require "spec_helper"
require "tempfile"
require "fileutils"

RSpec.describe "rux watch debouncing" do
  let(:temp_dir) { Dir.mktmpdir("rux-watch-debounce-test") }

  before do
    # Create a simple project structure
    FileUtils.mkdir_p("#{temp_dir}/lib")
    FileUtils.mkdir_p("#{temp_dir}/spec")

    File.write("#{temp_dir}/lib/fast_change.rb", "class FastChange; end")
    File.write("#{temp_dir}/spec/fast_change_spec.rb", <<~RUBY)
      RSpec.describe 'FastChange' do
        it 'passes' do
          expect(1).to eq(1)
        end
      end
    RUBY
  end

  after do
    FileUtils.rm_rf(temp_dir)
  end

  it "shows configurable debounce delay" do
    Dir.chdir(temp_dir) do
      output = `timeout 1 GO_LOG=debug rux watch --debounce 200 2>&1`
      expect(output).to include("Debounce delay: 200ms")
    end
  end

  it "uses default debounce of 100ms when not specified" do
    Dir.chdir(temp_dir) do
      output = `timeout 1 rux watch 2>&1`
      expect(output).to include("Debounce delay: 100ms")
    end
  end

  it "debounces rapid file changes" do
    # This test would ideally simulate rapid file changes and verify
    # that only one test run occurs, but it's difficult to test
    # timing-dependent behavior reliably in integration tests
    pending "Difficult to test timing-dependent behavior in integration tests"
  end
end
