require "spec_helper"
require "timeout"

RSpec.describe "rux watch signal handling" do
  include RuxWatchHelper

  it "runs indefinitely until receiving SIGINT signal" do
    skip "Signal handling test is flaky in CI environments"
  end

  it "exits immediately if spec directory doesn't exist" do
    Dir.mktmpdir do |tmpdir|
      result = run_rux_watch(dir: tmpdir, timeout: 1)

      expect(result.success?).to be false
      expect(result.err).to match(/no directories to watch found/i)
    end
  end
end
