# frozen_string_literal: true

require 'minitest/autorun'

# One of each non-passing outcome, each carrying a unique greppable token, so
# specs assert on exact content instead of patterns. Progress contribution:
# two F, one E, one S.
class OutcomesTest < Minitest::Test
  def test_failing_assertion
    assert_equal "TOKEN_WANTED", "TOKEN_GOT"
  end

  def test_raising_error
    raise ArgumentError, "TOKEN_BOOM"
  end

  def test_skipped
    skip "TOKEN_SKIPPED"
  end

  def test_output_then_failure
    puts "OUT_B4_KABOOM"
    flunk "TOKEN_FLUNKED"
  end
end
