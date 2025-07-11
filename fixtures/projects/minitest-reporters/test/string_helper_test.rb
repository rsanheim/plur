require_relative 'test_helper'

class StringHelper
  def self.reverse(str)
    str.reverse
  end

  def self.capitalize_words(str)
    str.split.map(&:capitalize).join(' ')
  end

  def self.remove_vowels(str)
    str.gsub(/[aeiouAEIOU]/, '')
  end
end

class StringHelperTest < Minitest::Test
  def test_reverse
    assert_equal "olleh", StringHelper.reverse("hello")
    assert_equal "dlrow", StringHelper.reverse("world")
  end

  def test_capitalize_words
    assert_equal "Hello World", StringHelper.capitalize_words("hello world")
    assert_equal "Ruby Is Great", StringHelper.capitalize_words("ruby is great")
  end

  def test_remove_vowels
    assert_equal "hll", StringHelper.remove_vowels("hello")
    assert_equal "wrld", StringHelper.remove_vowels("world")
  end
end