require_relative '../../../lib/example/processor'

RSpec.describe Example::Processor do
  it "processes" do
    expect(Example::Processor.new.process).to eq("Processing")
  end
end
