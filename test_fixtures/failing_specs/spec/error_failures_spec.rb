RSpec.describe 'ErrorFailures' do
  describe 'runtime errors' do
    it 'raises a standard error' do
      raise StandardError, 'Something went wrong!'
    end

    it 'raises a custom error' do
      raise 'Custom error message'
    end

    it 'has a NoMethodError' do
      nil.undefined_method
    end
  end

  describe 'syntax and name errors' do
    it 'has undefined variable' do
      undefined_variable
    end

    it 'divides by zero' do
      1 / 0
    end
  end
end