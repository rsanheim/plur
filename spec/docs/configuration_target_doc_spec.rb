require "spec_helper"

RSpec.describe "configuration target template documentation" do
  it "teaches automatic target appending and watch target placement" do
    doc = ROOT_PATH.join("docs", "configuration.md").read
    compact_doc = doc.gsub(/\s+/, " ")

    expect(doc).to include("Plur appends discovered targets automatically")
    expect(doc).to include("user job commands must not include")
    expect(doc).to include("Run mode rejects user job commands")
    expect(doc).to include("Watch mode can also use `{{target}}` inside a job command")
    expect(doc).to include("Configuration keys are strict")
    expect(doc).to include("job.rspec.cmdd")
    expect(compact_doc).to include("customize where resolved targets are placed")
    expect(compact_doc).to include("matching one-shot run mode")
  end
end
