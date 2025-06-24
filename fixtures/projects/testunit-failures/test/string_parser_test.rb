require 'test/unit'

class StringParser
  def self.extract_numbers(str)
    str.scan(/\d+/).map(&:to_i)
  end

  def self.word_frequency(text)
    words = text.downcase.split(/\W+/)
    words.each_with_object(Hash.new(0)) { |word, hash| hash[word] += 1 }
  end

  def self.parse_csv_line(line)
    line.split(',').map(&:strip)
  end

  def self.camel_to_snake(str)
    str.gsub(/([A-Z])/, '_\1').downcase.sub(/^_/, '')
  end
end

class TestStringParser < Test::Unit::TestCase
  def test_extract_numbers_success
    assert_equal [123, 456], StringParser.extract_numbers("abc123def456")
    assert_equal [], StringParser.extract_numbers("no numbers here")
  end

  def test_extract_numbers_with_decimals_failure
    # This will fail - doesn't handle decimals properly
    assert_equal [12, 34], StringParser.extract_numbers("12.34")
  end

  def test_word_frequency_simple
    result = StringParser.word_frequency("hello world hello")
    assert_equal 2, result["hello"]
    assert_equal 1, result["world"]
  end

  def test_word_frequency_case_sensitivity_failure
    result = StringParser.word_frequency("Hello WORLD hello")
    # This will fail - expects case sensitivity but method lowercases
    assert_equal 1, result["Hello"]
  end

  def test_parse_csv_success
    assert_equal ["a", "b", "c"], StringParser.parse_csv_line("a, b, c")
  end

  def test_parse_csv_with_quotes_failure
    # This will fail - doesn't handle quoted values
    assert_equal ["a,b", "c"], StringParser.parse_csv_line('"a,b", c')
  end

  def test_camel_to_snake_success
    assert_equal "hello_world", StringParser.camel_to_snake("HelloWorld")
    assert_equal "my_variable_name", StringParser.camel_to_snake("MyVariableName")
  end

  def test_camel_to_snake_edge_case_failure
    # This will fail - doesn't handle consecutive capitals well
    assert_equal "html_parser", StringParser.camel_to_snake("HTMLParser")
  end

  def test_nil_input_error
    # This will cause an error - no nil handling
    assert_equal [], StringParser.extract_numbers(nil)
  end
end