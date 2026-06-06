require "spec_helper"

RSpec.describe "output contract documentation" do
  it "documents stable dry-run JSON and watch find contracts" do
    doc = ROOT_PATH.join("docs", "output-contracts.md").read
    compact_doc = doc.gsub(/\s+/, " ")

    expect(doc).to include("--dry-run-format=json")
    expect(doc).to include("warnings")
    expect(doc).to include("workers")
    expect(doc).to include("argv`: command argv; this is the canonical command field for scripts")
    expect(compact_doc).to include("env`: environment entries Plur adds or overrides, including configured job env")
    expect(compact_doc).to include("Each environment key appears at most once, and duplicate configured entries keep the final effective value")
    expect(doc).to include("shell`: quoted, copyable command string for humans")
    expect(doc).to include("[dry-run] Plan:")
    expect(doc).to include("[dry-run] Commands:")
    expect(doc).to include("Worker stderr is streamed to stderr")
    expect(doc).to include("exit=80")
    expect(compact_doc).to include("Do not parse `shell`; use `argv` and `env` when executing from a script")
    expect(compact_doc).to include("Command and configuration errors in JSON modes still write plain text to stderr")
    expect(compact_doc).to include("Plur uses exit code 1 for failed work and many planning/runtime errors")
    expect(compact_doc).to include("Runnable changes include the matched rule, existing target, and final command plan")
    expect(compact_doc).to include("Exit code 2 means no runnable target exists for that changed file, including when no watch mappings are configured")
    expect(compact_doc).to include("In JSON mode, no configured watch mappings use the same empty JSON shape and exit code 2")
    expect(compact_doc).to include("Ignored watch changes also exit 2 before planning and include the admission result")
    expect(doc).to include("\"exit_code\": 0")
    expect(doc).to include("\"argv\": [\"bundle\", \"exec\", \"rspec\", \"spec/calculator_spec.rb\"]")
    expect(doc).to include("\"shell\": \"bundle exec rspec spec/calculator_spec.rb\"")
    expect(doc).to include("\"admission\": {")
    expect(doc).to include("\"reason\": \"ignored\"")
    expect(compact_doc).to include("admission`: live watch admission result, present when a preview is rejected before planning")
    expect(compact_doc).to include("job_plans`: runnable command plans, one per job")
    expect(compact_doc).to include("job_plans[].argv` and `job_plans[].env` are the canonical command fields for scripts")
    expect(compact_doc).to include("errors`: planning errors, present only when a watch preview fails while building a plan")
    expect(doc).to include("watch find")
    expect(doc).to include("[watch] Command:")
    expect(doc).to include("[watch] Hint: add a [[watch]] mapping")
    expect(doc).to include("Exit code 2")
    expect(doc).to include("\"exit_code\": 2")
    expect(doc).to include("not the machine API")
  end

  it "is linked from the docs index" do
    index = ROOT_PATH.join("docs", "index.md").read

    expect(index).to include("[Output Contracts](output-contracts.md)")
  end
end
