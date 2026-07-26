require 'minitest/autorun'

# `print` with no trailing newline. The pieces stay on one line and are only
# complete once the run ends, so this lives in its own file where no other
# example can merge into that line.
class PrintPartialTest < Minitest::Test
  def test_print_pieces
    print "PARTIAL_A"
    print "PARTIAL_B"
    assert true
  end
end
