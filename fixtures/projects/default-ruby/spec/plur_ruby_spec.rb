require "spec_helper"

RSpec.describe PlurRuby do
  describe ".awesome?" do
    it "returns true" do
      expect(PlurRuby.awesome?).to be true
    end
  end

  describe ".fast?" do
    it "returns true" do
      expect(PlurRuby.fast?).to be true
    end
  end

  describe ".version" do
    it "returns the version string" do
      expect(PlurRuby.version).to eq("0.1.0")
    end
  end
end
