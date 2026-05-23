require "spec_helper"

RSpec.describe "configuration target template documentation" do
  it "teaches run-mode targets as automatic and target templates as watch-only" do
    doc = ROOT_PATH.join("docs", "configuration.md").read

    expect(doc).to include("Plur appends discovered targets automatically")
    expect(doc).to include("user job commands must not include")
    expect(doc).to include("Run mode rejects user job commands")
    expect(doc).to include("Watch mode can also use `{{target}}` inside a job command")
    expect(doc).to include("In watch mode only")
  end
end
