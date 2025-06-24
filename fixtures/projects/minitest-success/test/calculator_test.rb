require 'minitest/autorun'

class Calculator
  def add(a, b)
    a + b
  end

  def subtract(a, b)
    a - b
  end

  def multiply(a, b)
    a * b
  end

  def divide(a, b)
    raise ArgumentError, "Cannot divide by zero" if b == 0
    a.to_f / b
  end
end

class CalculatorTest < Minitest::Test
  def setup
    @calculator = Calculator.new
  end

  def test_addition
    puts "in test_addition"
    assert_equal 5, @calculator.add(2, 3)
    assert_equal 0, @calculator.add(-1, 1)
    assert_equal -5, @calculator.add(-2, -3)
  end

  def test_subtraction
    assert_equal -1, @calculator.subtract(2, 3)
    assert_equal 5, @calculator.subtract(8, 3)
    assert_equal 0, @calculator.subtract(5, 5)
  end

  def test_multiplication
    assert_equal 6, @calculator.multiply(2, 3)
    assert_equal 0, @calculator.multiply(0, 100)
    assert_equal -12, @calculator.multiply(-3, 4)
  end

  def test_division
    assert_equal 2.0, @calculator.divide(6, 3)
    assert_equal 2.5, @calculator.divide(5, 2)
    assert_in_delta 0.333, @calculator.divide(1, 3), 0.001
  end

  def test_division_by_zero_raises_error
    assert_raises(ArgumentError) { @calculator.divide(10, 0) }
  end
end