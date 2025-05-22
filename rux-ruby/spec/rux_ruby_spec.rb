require "spec_helper"

RSpec.describe RuxRuby do
  describe ".awesome?" do
    it "returns true" do
      expect(RuxRuby.awesome?).to be true
    end
  end

  describe ".fast?" do
    it "returns true" do
      expect(RuxRuby.fast?).to be true
    end
  end

  describe ".version" do
    it "returns the version string" do
      expect(RuxRuby.version).to eq("0.1.0")
    end
  end
end