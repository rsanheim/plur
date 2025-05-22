require "spec_helper"
require_relative "../lib/date_formatter"

RSpec.describe DateFormatter do
  let(:test_date) { Date.new(2025, 5, 22) }

  describe ".format_iso" do
    it "formats date in ISO format" do
      expect(DateFormatter.format_iso(test_date)).to eq("2025-05-22")
    end
  end

  describe ".format_us" do
    it "formats date in US format" do
      expect(DateFormatter.format_us(test_date)).to eq("05/22/2025")
    end
  end

  describe ".format_long" do
    it "formats date in long format" do
      expect(DateFormatter.format_long(test_date)).to eq("May 22, 2025")
    end
  end

  describe ".days_between" do
    it "calculates days between dates" do
      date1 = Date.new(2025, 5, 20)
      date2 = Date.new(2025, 5, 22)
      expect(DateFormatter.days_between(date1, date2)).to eq(2)
    end
  end
end