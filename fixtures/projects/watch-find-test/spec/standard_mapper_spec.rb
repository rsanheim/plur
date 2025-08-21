require "spec_helper"

RSpec.describe StandardMapper do
  it "maps correctly" do
    expect(StandardMapper.new.map).to eq("standard")
  end
end