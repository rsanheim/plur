RSpec.describe 'MixedResults' do
  describe 'some pass, some fail' do
    it 'passes this test' do
      expect(2 + 2).to eq(4)
    end

    it 'fails this test' do
      expect('foo').to eq('bar')
    end

    it 'passes another test' do
      expect(true).to be_truthy
    end

    it 'has an error' do
      raise 'Unexpected error'
    end

    it 'passes one more' do
      expect([1, 2, 3]).to include(2)
    end
  end

  describe 'pending examples' do
    it 'is pending' do
      pending 'Not implemented yet'
      fail
    end

    xit 'is skipped' do
      expect(1).to eq(2)
    end

    it 'is also pending', pending: 'Waiting for feature' do
      expect(true).to be false
    end
  end
end