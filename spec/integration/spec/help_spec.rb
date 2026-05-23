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
    expect(result.out).to include("--dry-run")
    expect(result.out).to include("One-shot run preview only; watch mode rejects it")
    expect(result.out).to include("One-shot dry-run output format: text or json")
  end

  it "keeps watch find help focused on preview-specific flags" do
    result = run_plur("watch", "find", "--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur watch find <file-path> [flags]")
    expect(result.out).to include("--format")
    expect(result.out).to include("Output format: text or json")
    expect(result.out).not_to include("--dry-run")
    expect(result.out).not_to include("--dry-run-format")
    expect(result.out).not_to include("--json")
    expect(result.out).not_to include("--first-is-1")
    expect(result.out).not_to include("--workers")
    expect(result.out).not_to include("--rspec-split")
    expect(result.out).not_to include("--ignore")
  end
end
