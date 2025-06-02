# A characterization test library for CLIs
# Imagine something like [vcr](https://github.com/vcr/vcr), but for CLIs.
# See also https://en.wikipedia.org/wiki/Characterization_test 
#
#
# Example usage:
#
# it "records ruby system calls from within the 'record' block"
#   StayGold.record(record_as: "echo_hello") do
#     Open3.capture3("echo hello")
#   end
# end
#
# => records output of capture3 to a yaml file, along with metadata
#

module StayGold
  def self.record(record_as: nil)
  end

end