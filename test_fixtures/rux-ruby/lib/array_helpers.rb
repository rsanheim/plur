class ArrayHelpers
  def self.sum(array)
    array.sum
  end

  def self.average(array)
    return 0 if array.empty?
    array.sum.to_f / array.length
  end

  def self.max(array)
    array.max
  end

  def self.min(array)
    array.min
  end

  def self.unique_count(array)
    array.uniq.length
  end
end
