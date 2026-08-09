require "minitest/autorun"
require "stringio"

# Meta-test: builds and drives its own CompositeReporter the way reporter
# libraries and minitest's own suite do. Plur must leave test-constructed
# composites untouched (only the top-level composite gets plur's reporter).
class CompositeIntegrityTest < Minitest::Test
  def test_test_owned_composite_reporter_keeps_its_reporters
    io = StringIO.new(+"")
    composite = Minitest::CompositeReporter.new
    composite << Minitest::SummaryReporter.new(io)
    composite << Minitest::ProgressReporter.new(io)

    before = composite.reporters.map { |r| r.class.name }
    composite.start
    after = composite.reporters.map { |r| r.class.name }
    composite.report

    assert_equal before, after, "test-owned CompositeReporter lost its reporters"
    refute_empty io.string, "test-owned reporters captured no output"
  end

  def test_sanity
    assert_equal 2, 1 + 1
  end
end
