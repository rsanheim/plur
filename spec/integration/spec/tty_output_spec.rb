require "spec_helper"

# The TTY side of auto color resolution: under a real terminal, plur colors by
# default and NO_COLOR turns it off. Everything else lives in
# colorized_output_spec.rb, where pipes (the agent/CI environment) need no PTY.
RSpec.describe "Color resolution under a TTY", :pty do
  def run_passing_in_pty(env: {})
    fixture = project_fixture("default-ruby")
    run_in_pty(plur_binary, "-n", "2", "spec/calculator_spec.rb", chdir: fixture, env: env)
  end

  it "auto emits colored dots on a terminal" do
    output = run_passing_in_pty
    expect(output).to include("\e[32m.\e[0m")
  end

  it "NO_COLOR disables color even on a terminal" do
    output = run_passing_in_pty(env: {"NO_COLOR" => "1"})
    expect(output).not_to match(/\e\[\d+m/)
  end

  it "--color=never disables color even on a terminal" do
    fixture = project_fixture("default-ruby")
    output = run_in_pty(plur_binary, "--color=never", "-n", "2", "spec/calculator_spec.rb", chdir: fixture)
    expect(output).not_to match(/\e\[\d+m/)
  end
end
