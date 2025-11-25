require_relative '../lib/example'

RSpec.describe Example do
  it "greets" do
    expect(Example.new.greet).to eq("Hello")
  end
end
