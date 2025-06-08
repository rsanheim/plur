require "spec_helper"

RSpec.describe "ExampleScenarios" do
  describe "basic expectations" do
    it "passes with a simple expectation" do
      expect(1 + 1).to eq(2)
    end

    it "passes with a string comparison" do
      expect("hello").to eq("hello")
    end

    it "passes this one" do
      expect(true).to be true
    end
  end

  describe "errors and exceptions" do
    it "handles nil gracefully" do
      expect(nil).to be_nil
    end

    it "catches expected errors" do
      expect { raise StandardError, "Something went wrong!" }.to raise_error(StandardError, "Something went wrong!")
    end

    it "passes this one too" do
      expect([1, 2, 3]).to include(2)
    end
  end

  describe "array and hash operations" do
    it "passes on array contents" do
      expect([1, 2, 3]).to eq([1, 2, 3])
    end

    it "passes on hash contents" do
      expect({a: 1, b: 2}).to eq({a: 1, b: 2})
    end
  end
end
