require "spec_helper"

RSpec.describe "plur watch dry-run" do
  it "rejects dry-run watch with watch preview guidance" do
    stdout, stderr, status = Open3.capture3(
      plur_binary,
      "--dry-run",
      "watch",
      "--timeout",
      "1",
      chdir: default_ruby_dir.to_s
    )

    expect(status).not_to be_success
    expect(stdout).to eq("")
    expect(stderr).to include("plur watch does not support --dry-run yet")
    expect(stderr).to include("plur watch find <changed-file>")
    expect(stderr).not_to include("ready and watching")
  end
end
