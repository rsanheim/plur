require 'minitest/autorun'

module StringHelper
  def self.titleize(str)
    str.split(' ').map(&:capitalize).join(' ')
  end

  def self.reverse_words(str)
    str.split(' ').reverse.join(' ')
  end

  def self.count_vowels(str)
    str.downcase.count('aeiou')
  end
end

class StringHelperTest < Minitest::Test
  def test_titleize
    puts "in test_titleize"
    assert_equal "Hello World", StringHelper.titleize("hello world")
    assert_equal "Ruby Programming Language", StringHelper.titleize("ruby programming language")
    assert_equal "A", StringHelper.titleize("a")
  end

  def test_reverse_words
    assert_equal "world hello", StringHelper.reverse_words("hello world")
    assert_equal "test a is this", StringHelper.reverse_words("this is a test")
    assert_equal "word", StringHelper.reverse_words("word")
  end

  def test_count_vowels
    assert_equal 2, StringHelper.count_vowels("hello")
    assert_equal 5, StringHelper.count_vowels("aeiou")
    assert_equal 0, StringHelper.count_vowels("xyz")
    assert_equal 2, StringHelper.count_vowels("HELLO")
  end
end