require_relative '../../../lib/example/runner'

RSpec.describe Example::Runner do
  it "runs" do
    expect(Example::Runner.new.run).to eq("Running")
  end
end
