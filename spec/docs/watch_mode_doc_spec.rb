require "spec_helper"

RSpec.describe "watch mode documentation" do
  it "points previews to watch find instead of unsupported watch dry-run" do
    doc = ROOT_PATH.join("docs", "features", "watch-mode.md").read

    expect(doc).not_to include("plur watch --dry-run")
    expect(doc).to include("plur watch find")
  end
end
