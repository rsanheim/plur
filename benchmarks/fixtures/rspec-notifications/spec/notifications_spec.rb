# A fixed, inexpensive suite isolates Plur's notification processing from the
# application work performed by the larger pinned benchmark projects.
RSpec.describe "notification throughput" do
  20_000.times do |index|
    it("collects example #{index}") {}
  end
end
