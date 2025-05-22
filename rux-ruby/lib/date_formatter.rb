require "date"

class DateFormatter
  def self.format_iso(date)
    date.strftime("%Y-%m-%d")
  end

  def self.format_us(date)
    date.strftime("%m/%d/%Y")
  end

  def self.format_long(date)
    date.strftime("%B %d, %Y")
  end

  def self.days_between(date1, date2)
    (date2 - date1).to_i
  end
end