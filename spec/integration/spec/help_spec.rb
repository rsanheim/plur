require "spec_helper"

RSpec.describe "help output" do
  it "leads top-level help with commandless usage and examples" do
    result = run_plur("--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur [patterns...] [flags]")
    expect(result.out).to include("plur <command> [flags]")
    expect(result.out).to include("A fast, parallel test runner and watcher for Ruby/RSpec")
    expect(result.out).to include("Examples:")
    expect(result.out).to include("Examples:\n    plur")
    expect(result.out).to include("plur spec/calculator_spec.rb")
    expect(result.out).to include("plur --dry-run")
    expect(result.out).to include("plur watch find spec/calculator_spec.rb")
    expect(result.out).not_to include("--rspec-split")
    expect(result.out).to include("Daily commands")
    expect(result.out).to include("Advanced and setup commands")
    expect(result.out.index("Daily commands")).to be < result.out.index("Advanced and setup commands")

    daily_commands = result.out[/Daily commands.*?(?=Advanced and setup commands)/m]
    advanced_commands = result.out[/Advanced and setup commands.*?(?=Flags:)/m]

    expect(daily_commands).to include("spec")
    expect(daily_commands).to include("watch run")
    expect(daily_commands).to include("watch find")
    expect(daily_commands).not_to include("watch install")

    expect(advanced_commands).to include("watch install")
    expect(advanced_commands).to include("rails (rake)")
    expect(advanced_commands).to include("doctor")
    expect(advanced_commands).to include("config init")
    expect(advanced_commands).to include("rails:init")
    expect(advanced_commands).to include("version")
  end

  it "leads watch help with default watch mode and preview workflows" do
    result = run_plur("watch", "--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur watch [flags]")
    expect(result.out).to include("plur watch find <changed-file> [flags]")
    expect(result.out).to include("Examples:")
    expect(result.out).to include("Examples:\n    plur watch")
    expect(result.out).to include("plur watch")
    expect(result.out).to include("plur watch find spec/calculator_spec.rb")
    expect(result.out).to include("Daily commands")
    expect(result.out).to include("Advanced and setup commands")
    expect(result.out.index("Daily commands")).to be < result.out.index("Advanced and setup commands")
    expect(result.out.index("watch find")).to be < result.out.index("watch install")
    expect(result.out).to include("--ignore")
    expect(result.out).to include("Patterns to ignore from watch events")
    expect(result.out).not_to match(/^\s+--dry-run\s/m)
    expect(result.out).not_to include("--json")
    expect(result.out).not_to include("--first-is-1")
    expect(result.out).not_to include("--workers")
    expect(result.out).not_to include("--rspec-split")
  end

  it "keeps watch run help focused on live-watch controls" do
    result = run_plur("watch", "run", "--help")

    expect(result).to be_success
    expect(result.out).to include("--ignore")
    expect(result.out).to include("--timeout")
    expect(result.out).to include("--debounce")
    expect(result.out).to include("--use")
    expect(result.out).not_to match(/^\s+--dry-run\s/m)
    expect(result.out).not_to include("--json")
    expect(result.out).not_to include("--first-is-1")
    expect(result.out).not_to include("--workers")
    expect(result.out).not_to include("--rspec-split")
  end

  it "keeps watch find help focused on preview inputs" do
    result = run_plur("watch", "find", "--help")

    expect(result).to be_success
    expect(result.out).to include("Usage: plur watch find <file-path> [flags]")
    expect(result.out).to include("--ignore")
    expect(result.out).to include("Patterns to ignore from watch events")
    expect(result.out).not_to match(/^\s+--dry-run\s/m)
    expect(result.out).not_to include("--json")
    expect(result.out).not_to include("--first-is-1")
    expect(result.out).not_to include("--workers")
    expect(result.out).not_to include("--rspec-split")
  end

  it "leaves spec help with run-mode flags" do
    result = run_plur("spec", "--help")

    expect(result).to be_success
    expect(result.out).to include("--dry-run")
    expect(result.out).to include("--workers")
    expect(result.out).not_to include("--json")
    expect(result.out).to include("--rspec-split")
  end
end
