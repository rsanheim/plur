require "spec_helper"
require_relative "../lib/validator"

RSpec.describe Validator do
  describe ".email?" do
    it "validates correct email" do
      expect(Validator.email?("test@example.com")).to be true
    end

    it "rejects invalid email" do
      expect(Validator.email?("invalid-email")).to be false
    end
  end

  describe ".phone?" do
    it "validates correct phone format" do
      expect(Validator.phone?("123-456-7890")).to be true
    end

    it "rejects invalid phone format" do
      expect(Validator.phone?("1234567890")).to be false
    end
  end

  describe ".positive_integer?" do
    it "validates positive integers" do
      expect(Validator.positive_integer?(5)).to be true
    end

    it "rejects negative numbers" do
      expect(Validator.positive_integer?(-1)).to be false
    end

    it "rejects non-integers" do
      expect(Validator.positive_integer?(5.5)).to be false
    end
  end

  describe ".in_range?" do
    it "validates numbers in range" do
      expect(Validator.in_range?(5, 1, 10)).to be true
    end

    it "rejects numbers out of range" do
      expect(Validator.in_range?(15, 1, 10)).to be false
    end
  end
end
