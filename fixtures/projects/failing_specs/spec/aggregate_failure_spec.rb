# A realistic-looking spec file mixing passing examples, plain failures, and an
# aggregate-failures example. Ordered (RSpec runs in definition order) so the
# failure numbering exercises the interesting case: a plain failure (1), an
# aggregate failure whose sub-failures inherit its number (2 -> 2.1/2.2/2.3),
# then another plain failure (3) — proving the top-level counter keeps
# incrementing past the aggregate's sub-markers.
RSpec.describe "Order" do
  it "sums the line item prices into a subtotal" do
    prices = [1099, 500, 250]
    expect(prices.sum).to eq(1849)
  end

  it "counts the items in the cart" do
    expect([:book, :pen, :mug].length).to eq(3)
  end

  it "formats the subtotal as currency" do
    expect(format("$%.2f", 1849 / 100.0)).to eq("$18.49")
  end

  it "adds sales tax to the subtotal" do
    subtotal = 1849
    tax = (subtotal * 0.08).round
    expect(subtotal + tax).to eq(2000) # 1849 + 148 = 1997
  end

  it "validates every order field at once", :aggregate_failures do
    expect(1).to eq(2)
    expect("pending").to eq("shipped")
    expect([]).not_to be_empty
  end

  it "accepts only numeric coupon codes" do
    expect("SAVE10".match?(/\A\d+\z/)).to be(true)
  end
end
