require 'minitest/autorun'

class ArrayOperations
  def self.sum(arr)
    arr.reduce(0, :+)
  end

  def self.average(arr)
    return 0 if arr.empty?
    sum(arr) / arr.length.to_f
  end

  def self.find_max(arr)
    arr.max
  end

  def self.unique_elements(arr)
    arr.uniq
  end
end

class ArrayOperationsTest < Minitest::Test
  def test_sum_with_positive_numbers
    assert_equal 15, ArrayOperations.sum([1, 2, 3, 4, 5])
  end

  def test_sum_with_empty_array
    assert_equal 0, ArrayOperations.sum([])
  end

  def test_average_calculation_failure
    # This will fail - wrong expected value
    assert_equal 3, ArrayOperations.average([1, 2, 3, 4])
  end

  def test_find_max_with_nil
    # This will cause an error - doesn't handle nil
    assert_equal 10, ArrayOperations.find_max([1, nil, 10, 5])
  end

  def test_unique_elements_success
    assert_equal [1, 2, 3], ArrayOperations.unique_elements([1, 2, 2, 3, 3, 3])
  end

  def test_average_precision_failure
    # This will fail - float comparison without delta
    assert_equal 2.33, ArrayOperations.average([1, 2, 4])
  end
end