require "spec_helper"

RSpec.describe "Backspin edge cases" do

  it "raises error for empty dubplate name" do
    expect {
      Backspin.record("") do
        Open3.capture3("echo empty")
      end
    }.to raise_error(ArgumentError, "dubplate_name is required")
  end

  it "uses provided dubplate name" do
    result = Backspin.record("custom_name") do
      Open3.capture3("echo custom")
    end

    expect(result.dubplate_path.to_s).to end_with("custom_name.yaml")
  end

  it "sanitizes dubplate names with special characters" do
    result = Backspin.record("test/with/slashes") do
      Open3.capture3("echo slashes")
    end

    # Slashes should create subdirectories
    expect(result.dubplate_path.to_s).to end_with("test/with/slashes.yaml")
  end

end
