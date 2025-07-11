require_relative 'test_helper'

class DeprecationTest < Minitest::Test
  def test_with_deprecation_warning
    # Simulate a deprecation warning going to stderr
    warn "DEPRECATION WARNING: This is a test deprecation warning"
    assert_equal 1, 1
  end

  def test_another_with_multiline_warning
    warn "DEPRECATION WARNING: This is a multiline deprecation\nwith additional context\n(called from somewhere)"
    assert_equal 2, 2
  end
  
  def test_normal_output
    puts "This is normal stdout output"
    assert true
  end
end