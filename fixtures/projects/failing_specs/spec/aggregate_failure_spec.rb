RSpec.describe "Aggregate Failure" do
  it "reports several failures at once", :aggregate_failures do
    expect(1).to eq(2)
    expect("foo").to eq("bar")
    expect([1, 2, 3]).to be_empty
  end
end
