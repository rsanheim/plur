require "spec_helper"
require_relative "../lib/counter"

RSpec.describe Counter do
  let(:counter) { Counter.new }

  describe "#initialize" do
    it "starts at zero by default" do
      expect(counter.value).to eq(0)
    end

    it "accepts a starting value" do
      counter = Counter.new(5)
      expect(counter.value).to eq(5)
    end
  end

  describe "#increment" do
    it "increases value by 1" do
      counter.increment
      expect(counter.value).to eq(1)
    end
  end

  describe "#decrement" do
    it "decreases value by 1" do
      counter.increment
      counter.decrement
      expect(counter.value).to eq(0)
    end
  end

  describe "#reset" do
    it "resets to zero" do
      counter.increment
      counter.increment
      counter.reset
      expect(counter.value).to eq(0)
    end
  end

  describe "#even?" do
    it "returns true for even values" do
      counter.increment
      counter.increment
      expect(counter).to be_even
    end
  end

  describe "#odd?" do
    it "returns true for odd values" do
      counter.increment
      expect(counter).to be_odd
    end
  end
end