require "spec_helper"

RSpec.describe Utils::Formatter do
  it "formats text" do
    expect(Utils::Formatter.new.format("hello")).to eq("HELLO")
  end
end