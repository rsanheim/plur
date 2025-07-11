require 'test/unit'

class MathOperations
  def self.factorial(n)
    return 1 if n <= 1
    n * factorial(n - 1)
  end

  def self.fibonacci(n)
    return n if n <= 1
    fibonacci(n - 1) + fibonacci(n - 2)
  end

  def self.is_prime?(n)
    return false if n < 2
    (2..Math.sqrt(n)).none? { |i| n % i == 0 }
  end
end

class TestMathOperations < Test::Unit::TestCase
  def test_factorial
    assert_equal 1, MathOperations.factorial(0)
    assert_equal 1, MathOperations.factorial(1)
    assert_equal 120, MathOperations.factorial(5)
    assert_equal 3628800, MathOperations.factorial(10)
  end

  def test_fibonacci
    assert_equal 0, MathOperations.fibonacci(0)
    assert_equal 1, MathOperations.fibonacci(1)
    assert_equal 5, MathOperations.fibonacci(5)
    assert_equal 55, MathOperations.fibonacci(10)
  end

  def test_is_prime
    assert_true MathOperations.is_prime?(2)
    assert_true MathOperations.is_prime?(3)
    assert_true MathOperations.is_prime?(17)
    assert_false MathOperations.is_prime?(1)
    assert_false MathOperations.is_prime?(4)
    assert_false MathOperations.is_prime?(15)
  end
end