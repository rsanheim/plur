require 'spec_helper'

RSpec.describe 'FailingExamples' do
  describe 'expectation failures' do
    it 'fails with a simple expectation' do
      expect(1 + 1).to eq(3)
    end

    it 'fails with a string comparison' do
      expect('hello').to eq('goodbye')
    end

    it 'passes this one' do
      expect(true).to be true
    end
  end

  describe 'errors and exceptions' do
    it 'raises a NoMethodError' do
      nil.this_method_does_not_exist
    end

    it 'raises a custom error' do
      raise StandardError, 'Something went wrong!'
    end

    it 'passes this one too' do
      expect([1, 2, 3]).to include(2)
    end
  end

  describe 'array and hash failures' do
    it 'fails on array contents' do
      expect([1, 2, 3]).to eq([1, 2, 4])
    end

    it 'fails on hash contents' do
      expect({ a: 1, b: 2 }).to eq({ a: 1, b: 3 })
    end
  end
end