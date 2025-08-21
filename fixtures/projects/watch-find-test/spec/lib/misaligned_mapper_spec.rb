require "spec_helper"

RSpec.describe MisalignedMapper do
  it "maps in misaligned location" do
    expect(MisalignedMapper.new.map).to eq("misaligned")
  end
end