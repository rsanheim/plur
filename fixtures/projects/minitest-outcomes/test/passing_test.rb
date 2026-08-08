# frozen_string_literal: true

require 'minitest/autorun'

# All tests here pass. Two write to stdout mid-run, so their text lands on the
# same physical line as minitest's progress characters - the interleaving a
# runner has to untangle.
#
# NOTE: every token written to stdout during the run avoids the characters
# ".", "F", "E", and "S", so specs can count progress characters exactly in
# raw minitest output without heuristics.
class PassingTest < Minitest::Test
  def test_addition
    assert_equal 4, 2 + 2
  end

  def test_inclusion
    assert_includes [1, 2, 3], 2
  end

  def test_not_nil
    refute_nil "value"
  end

  def test_pass_with_puts
    puts "OUT_MID_RUN"
    assert true
  end

  def test_pass_via_stdout_constant
    STDOUT.puts "OUT_GLOBAL_IO"
    assert true
  end
end
