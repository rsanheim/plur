class Validator
  def self.email?(email)
    email.match?(/\A[\w+\-.]+@[a-z\d\-]+(\.[a-z\d\-]+)*\.[a-z]+\z/i)
  end

  def self.phone?(phone)
    phone.match?(/\A\d{3}-\d{3}-\d{4}\z/)
  end

  def self.positive_integer?(num)
    num.is_a?(Integer) && num > 0
  end

  def self.in_range?(num, min, max)
    num >= min && num <= max
  end
end