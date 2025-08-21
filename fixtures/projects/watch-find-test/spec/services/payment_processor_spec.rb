require "spec_helper"

RSpec.describe PaymentProcessor do
  it "processes payments" do
    expect(PaymentProcessor.new.process).to eq("processing payment")
  end
end