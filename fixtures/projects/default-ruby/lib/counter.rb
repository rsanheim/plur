class Counter
  def initialize(start = 0)
    @value = start
  end

  def increment
    @value += 1
  end

  def decrement
    @value -= 1
  end

  def reset
    @value = 0
  end

  attr_reader :value

  def even?
    @value.even?
  end

  def odd?
    @value.odd?
  end
end
