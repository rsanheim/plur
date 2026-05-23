require "spec_helper"

RSpec.describe "output contract documentation" do
  it "documents stable dry-run JSON and watch find contracts" do
    doc = ROOT_PATH.join("docs", "output-contracts.md").read
    compact_doc = doc.gsub(/\s+/, " ")

    expect(doc).to include("--dry-run-format=json")
    expect(doc).to include("warnings")
    expect(doc).to include("workers")
    expect(doc).to include("argv`: command argv; this is the canonical command field for scripts")
    expect(doc).to include("shell`: quoted, copyable command string for humans")
    expect(doc).to include("[dry-run] Plan:")
    expect(doc).to include("[dry-run] Commands:")
    expect(doc).to include("Worker stderr is streamed to stderr")
    expect(doc).to include("exit=80")
    expect(compact_doc).to include("Do not parse `shell`; use `argv` and `env` when executing from a script")
    expect(compact_doc).to include("Command and configuration errors in JSON modes still write plain text to stderr")
    expect(compact_doc).to include("Plur uses exit code 1 for failed work and many planning/runtime errors")
    expect(doc).to include("watch find")
    expect(doc).to include("Exit code 2")
    expect(doc).to include("\"exit_code\": 2")
    expect(doc).to include("not the machine API")
  end

  it "is linked from the docs index" do
    index = ROOT_PATH.join("docs", "index.md").read

    expect(index).to include("[Output Contracts](output-contracts.md)")
  end
end
