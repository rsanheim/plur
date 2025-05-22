require "spec_helper"
require_relative "../lib/array_helpers"

RSpec.describe ArrayHelpers do
  describe ".sum" do
    it "sums an array of numbers" do
      expect(ArrayHelpers.sum([1, 2, 3])).to eq(6)
    end

    it "handles empty arrays" do
      expect(ArrayHelpers.sum([])).to eq(0)
    end
  end

  describe ".average" do
    it "calculates average of numbers" do
      expect(ArrayHelpers.average([1, 2, 3])).to eq(2.0)
    end

    it "handles empty arrays" do
      expect(ArrayHelpers.average([])).to eq(0)
    end
  end

  describe ".max" do
    it "finds maximum value" do
      expect(ArrayHelpers.max([1, 5, 3])).to eq(5)
    end
  end

  describe ".min" do
    it "finds minimum value" do
      expect(ArrayHelpers.min([1, 5, 3])).to eq(1)
    end
  end

  describe ".unique_count" do
    it "counts unique elements" do
      expect(ArrayHelpers.unique_count([1, 2, 2, 3])).to eq(3)
    end
  end
end