require "spec_helper"

RSpec.describe "help output" do
  it "leads top-level help with commandless usage and common workflows" do
    result = run_plur("--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur [patterns...] [flags]")
    expect(result.out).to include("plur <command> [flags]")
    expect(result.out).to include("Common workflows:")
    expect(result.out).to include("plur spec/calculator_spec.rb")
    expect(result.out).to include("plur watch find spec/calculator_spec.rb")
  end

  it "leads watch help with default watch mode and watch find preview" do
    result = run_plur("watch", "--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur watch [flags]")
    expect(result.out).to include("plur watch find <changed-file> [flags]")
    expect(result.out).to include("Common workflows:")
    expect(result.out).to include("plur watch")
    expect(result.out).to include("plur watch find spec/calculator_spec.rb")
  end
end
