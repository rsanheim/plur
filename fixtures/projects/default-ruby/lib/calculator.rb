class Calculator
  def add(a, b)
    a + b
  end

  def multiply(a, b)
    a * b
  end

  def divide(a, b)
    raise ArgumentError, "Cannot divide by zero" if b == 0
    a / b.to_f
  end
end
# test
# test
# test
# test
# test
# test
# test
# test

# test
# test
# test
# test
# test
# test
