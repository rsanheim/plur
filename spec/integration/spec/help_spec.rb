require "spec_helper"

RSpec.describe "help output" do
  it "leads top-level help with commandless usage and common workflows" do
    result = run_plur("--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur [patterns...] [flags]")
    expect(result.out).to include("plur <command> [flags]")
    expect(result.out).to include("Common workflows:")
    expect(result.out).to include("plur spec/calculator_spec.rb")
    expect(result.out).to include("plur test/calculator_test.rb")
    expect(result.out).not_to match(/^  plur test\s{2,}Run Minitest targets$/)
    expect(result.out).to include("plur watch find spec/calculator_spec.rb")
    expect(result.out).to include("Daily commands")
    expect(result.out).to include("Advanced and setup commands")
    expect(result.out.index("Daily commands")).to be < result.out.index("Advanced and setup commands")

    daily_commands = result.out[/Daily commands.*?(?=Advanced and setup commands)/m]
    advanced_commands = result.out[/Advanced and setup commands.*/m]

    expect(daily_commands).to include("spec [<patterns> ...]")
    expect(daily_commands).to include("watch run")
    expect(daily_commands).to include("watch find <file-path>")
    expect(daily_commands).not_to include("watch install")

    expect(advanced_commands).to include("watch install")
    expect(advanced_commands).to include("rails (rake)")
    expect(advanced_commands).to include("doctor")
    expect(advanced_commands).to include("config init")
    expect(advanced_commands).to include("rails:init")
    expect(advanced_commands).to include("version")
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
    expect(result.out.index("Daily commands")).to be < result.out.index("Advanced and setup commands")
    expect(result.out.index("watch find <file-path>")).to be < result.out.index("watch install")
  end

  it "does not advertise the unused JSON file output flag" do
    [
      ["--help"],
      ["spec", "--help"],
      ["watch", "--help"],
      ["watch", "run", "--help"]
    ].each do |args|
      result = run_plur(*args)

      expect(result).to be_success
      expect(result.out).not_to include("--json")
      expect(result.out).not_to include("Save detailed test results as JSON")
    end
  end

  it "rejects the removed JSON file output flag" do
    result = run_plur_allowing_errors("--json=tmp/results.json", "--dry-run")

    expect(result.exit_status).to eq(1)
    expect(result.err).to include("Error: --json is not a Plur flag.")
    expect(result.err).to include("plur --dry-run --dry-run-format=json [patterns...]")
    expect(result.err).to include("plur watch find --format=json <file>")
    expect(result.err).not_to include("did you mean")
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

  it "does not show spec command help for a target named test" do
    examples = [
      ["test", "--help"],
      ["test", "-h"],
      ["-C", project_fixture("minitest-success").to_s, "test", "--help"]
    ]

    examples.each do |args|
      result = run_plur_allowing_errors(*args)

      expect(result.exit_status).to eq(1)
      expect(result.err).to include("Error: `test` is a target path, not a Plur command.")
      expect(result.err).to include("Use `plur test/calculator_test.rb` to run a Minitest target.")
      expect(result.out).not_to include("Usage: plur spec")
    end
  end
end
