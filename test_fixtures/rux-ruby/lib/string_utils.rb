class StringUtils
  def self.reverse(str)
    str.reverse
  end

  def self.upcase(str)
    str.upcase
  end

  def self.word_count(str)
    str.split.length
  end

  def self.palindrome?(str)
    cleaned = str.downcase.gsub(/[^a-z]/, "")
    cleaned == cleaned.reverse
  end
end
# test
# test
# test
# test
