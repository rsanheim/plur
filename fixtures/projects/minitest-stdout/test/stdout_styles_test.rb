require 'minitest/autorun'

# Every flavour of whole-line stdout a test can produce. Nothing here leaves an
# unterminated line, so the output is the same whatever order minitest picks.
class StdoutStylesTest < Minitest::Test
  def test_progress_lookalike_text
    puts "FIRST_LINE"
    assert true
  end

  def test_multiline_string
    puts "MULTI_ONE\nMULTI_TWO"
    assert true
  end

  def test_array_and_blank
    puts ["ARRAY_ONE", "ARRAY_TWO"]
    puts
    puts "AFTER_BLANK"
    assert true
  end

  def test_stdout_constant
    STDOUT.puts "VIA_STDOUT_CONSTANT"
    assert true
  end

  def test_marker_lookalike
    puts "PLUR_OUT:NOT_REALLY_A_MARKER"
    assert true
  end

  def test_dots_only
    puts "....."
    assert true
  end

  def test_print_then_newline
    print "TERMINATED_A"
    puts "TERMINATED_B"
    assert true
  end

  def test_very_long_line
    puts "LONG_LINE_START" + ("L" * 300_000)
    assert true
  end
end

# Writing through $stdout has to keep behaving like an IO.
class StdoutContractTest < Minitest::Test
  def test_write_returns_bytes_written
    assert_equal 6, $stdout.write("BYTES\n")
    assert_equal 3, $stdout.write("A", "B", "\n")
  end

  def test_binary_bytes_do_not_raise
    $stdout.write("BINARY:\xC0\xFF\n".b)
    assert true
  end

  def test_capture_io_nests
    out, _err = capture_io { puts "INSIDE_CAPTURE_IO" }
    assert_equal "INSIDE_CAPTURE_IO\n", out
    puts "AFTER_CAPTURE_IO"
  end

  def test_capture_subprocess_io_is_untouched
    out, _err = capture_subprocess_io do
      system("echo SUBPROCESS_LINE")
      puts "RUBY_LINE_INSIDE"
    end
    # No marker leaks into what the test captured
    assert_equal "SUBPROCESS_LINE\nRUBY_LINE_INSIDE\n", out
    puts "AFTER_CAPTURE_SUBPROCESS"
  end

  def test_manual_stdout_swap
    original = $stdout
    buffer = StringIO.new
    $stdout = buffer
    puts "INSIDE_MANUAL_SWAP"
    $stdout = original
    assert_equal "INSIDE_MANUAL_SWAP\n", buffer.string
    puts "AFTER_MANUAL_SWAP"
  end
end

# Threads writing at the same time must not interleave inside a line.
class ParallelOutputTest < Minitest::Test
  parallelize_me!

  4.times do |i|
    define_method("test_parallel_#{i}") do
      25.times { |j| puts "PARALLEL_#{i}_#{j}" }
      assert true
    end
  end
end
