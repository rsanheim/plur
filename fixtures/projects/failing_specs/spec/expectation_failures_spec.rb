RSpec.describe "ExpectationFailures" do
  describe "pending examples" do
    it "is pending in expectation_failures" do
      pending "Pending from expectation_failures_spec"
      fail
    end

    xit "is skipped in expectation_failures" do
      expect(1).to eq(2)
    end
  end

  describe "basic failures" do
    it "fails with incorrect math" do
      expect(1 + 1).to eq(3)
    end

    it "fails with string mismatch" do
      expect("hello").to eq("goodbye")
    end

    it "fails with false expectation" do
      expect(true).to be false
    end
  end

  describe "complex failures" do
    it "fails with array mismatch" do
      expect([1, 2, 3]).to eq([4, 5, 6])
    end

    it "fails with hash mismatch" do
      expect({a: 1, b: 2}).to eq({c: 3, d: 4})
    end

    it "fails with nil check" do
      expect(nil).not_to be_nil
    end
  end
end
