require 'minitest/autorun'

class StringHelperTest < Minitest::Test
  def test_upcase
    assert_equal "HELLO", "hello".upcase
  end

  def test_downcase
    assert_equal "world", "WORLD".downcase
  end

  def test_capitalize
    assert_equal "Ruby", "ruby".capitalize
  end
end
