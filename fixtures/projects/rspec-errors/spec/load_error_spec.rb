# LoadError - requiring non-existent file
require 'non_existent_file'

RSpec.describe "LoadError" do
  it "never runs" do
    expect(1).to eq(1)
  end
end