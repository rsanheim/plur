require 'minitest/autorun'
require 'minitest/reporters'

Minitest::Reporters.use!(Minitest::Reporters::DefaultReporter.new(color: true))

class SimpleTest < Minitest::Test
  def test_one
    assert_equal 1, 1
  end

  def test_two
    assert_equal 2, 2
  end
end