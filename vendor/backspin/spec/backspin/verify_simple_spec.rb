require "spec_helper"

RSpec.describe "Backspin simple verify" do
  let(:dubplate_dir) { Pathname.new(File.join(Dir.pwd, "..")).join("tmp", "backspin") }

  it "verifies matching output" do
    # Record
    Backspin.record("simple_echo") do
      Open3.capture3("echo hello")
    end

    # Verify - should pass
    result = Backspin.verify("simple_echo") do
      Open3.capture3("echo hello")
    end

    expect(result.verified?).to be true
  end

  it "detects non-matching output" do
    # Record
    Backspin.record("echo_original") do
      Open3.capture3("echo original")
    end

    # Verify - should fail
    result = Backspin.verify("echo_original") do
      Open3.capture3("echo different")
    end

    expect(result.verified?).to be false
  end

  it "raises error when dubplate not found" do
    expect {
      Backspin.verify("missing") do
        Open3.capture3("echo test")
      end
    }.to raise_error(Backspin::DubplateNotFoundError)
  end

  after do
    FileUtils.rm_rf(dubplate_dir)
  end
end
