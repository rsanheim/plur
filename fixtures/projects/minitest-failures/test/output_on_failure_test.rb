require 'minitest/autorun'

# Tests that write to stdout and then fail or error: the output has to survive
# alongside the failure details.
class OutputOnFailureTest < Minitest::Test
  def test_writes_then_fails
    puts "OUTPUT_BEFORE_FAILURE"
    assert_equal "expected", "actual"
  end

  def test_writes_then_raises
    puts "OUTPUT_BEFORE_ERROR"
    raise ArgumentError, "boom"
  end

  def test_writes_then_passes
    puts "OUTPUT_FROM_PASSING_TEST"
    assert true
  end
end
