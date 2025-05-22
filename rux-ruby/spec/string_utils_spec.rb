require "spec_helper"
require_relative "../lib/string_utils"

RSpec.describe StringUtils do
  describe ".reverse" do
    it "reverses a string" do
      expect(StringUtils.reverse("hello")).to eq("olleh")
    end

    it "handles empty strings" do
      expect(StringUtils.reverse("")).to eq("")
    end
  end

  describe ".upcase" do
    it "converts to uppercase" do
      expect(StringUtils.upcase("hello")).to eq("HELLO")
    end
  end

  describe ".word_count" do
    it "counts words in a string" do
      expect(StringUtils.word_count("hello world")).to eq(2)
    end

    it "handles single words" do
      expect(StringUtils.word_count("hello")).to eq(1)
    end
  end

  describe ".palindrome?" do
    it "detects palindromes" do
      expect(StringUtils.palindrome?("racecar")).to be true
    end

    it "detects non-palindromes" do
      expect(StringUtils.palindrome?("hello")).to be false
    end
  end
end