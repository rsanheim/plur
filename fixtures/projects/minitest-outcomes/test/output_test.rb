# frozen_string_literal: true

require 'minitest/autorun'

# stdout styles that stress line handling: an unterminated print, a print
# completed later on the same line, and a multi-line puts. All tests pass.
class OutputTest < Minitest::Test
  def test_print_without_newline
    print "PARTIAL_A"
    print "PARTIAL_B"
    assert true
  end

  def test_print_then_newline
    print "CHUNK_A"
    puts "CHUNK_B"
    assert true
  end

  def test_multiline_puts
    puts "MULTI_1\nMULTI_2"
    assert true
  end
end
