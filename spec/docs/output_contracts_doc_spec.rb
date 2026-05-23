require "spec_helper"

RSpec.describe "output contract documentation" do
  it "documents stable dry-run JSON and watch find contracts" do
    doc = ROOT_PATH.join("docs", "output-contracts.md").read

    expect(doc).to include("--dry-run-format=json")
    expect(doc).to include("warnings")
    expect(doc).to include("workers")
    expect(doc).to include("watch find")
    expect(doc).to include("Exit code 2")
    expect(doc).to include("not the machine API")
  end

  it "is linked from the docs index" do
    index = ROOT_PATH.join("docs", "index.md").read

    expect(index).to include("[Output Contracts](output-contracts.md)")
  end
end
