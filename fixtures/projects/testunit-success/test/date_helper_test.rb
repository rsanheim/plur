require 'test/unit'
require 'date'

class DateHelper
  def self.days_between(date1, date2)
    (date2 - date1).to_i.abs
  end

  def self.format_date(date, format = '%Y-%m-%d')
    date.strftime(format)
  end

  def self.weekend?(date)
    date.saturday? || date.sunday?
  end

  def self.add_business_days(date, days)
    result = date
    days.times do
      result = result.next_day
      result = result.next_day while weekend?(result)
    end
    result
  end
end

class TestDateHelper < Test::Unit::TestCase
  def setup
    @monday = Date.new(2024, 1, 1)    # Monday
    @friday = Date.new(2024, 1, 5)    # Friday
    @saturday = Date.new(2024, 1, 6)  # Saturday
    @sunday = Date.new(2024, 1, 7)    # Sunday
  end

  def test_days_between
    assert_equal 4, DateHelper.days_between(@monday, @friday)
    assert_equal 4, DateHelper.days_between(@friday, @monday)
    assert_equal 0, DateHelper.days_between(@monday, @monday)
  end

  def test_format_date
    assert_equal '2024-01-01', DateHelper.format_date(@monday)
    assert_equal '01/01/2024', DateHelper.format_date(@monday, '%m/%d/%Y')
    assert_equal 'January 01, 2024', DateHelper.format_date(@monday, '%B %d, %Y')
  end

  def test_weekend
    assert_false DateHelper.weekend?(@monday)
    assert_false DateHelper.weekend?(@friday)
    assert_true DateHelper.weekend?(@saturday)
    assert_true DateHelper.weekend?(@sunday)
  end

  def test_add_business_days
    # Adding 1 business day to Monday gives Tuesday
    assert_equal Date.new(2024, 1, 2), DateHelper.add_business_days(@monday, 1)
    
    # Adding 5 business days to Monday gives next Monday
    assert_equal Date.new(2024, 1, 8), DateHelper.add_business_days(@monday, 5)
    
    # Adding 1 business day to Friday gives next Monday
    assert_equal Date.new(2024, 1, 8), DateHelper.add_business_days(@friday, 1)
  end
end