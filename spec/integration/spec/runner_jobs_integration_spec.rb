require "spec_helper"

RSpec.describe "Runner jobs resolution integration" do
  let(:config_fixture_dir) { project_fixture("config-test") }
  let(:config_file) { config_fixture_dir.join("runner-jobs.toml") }

  it "autodetects rspec with cmd-only override and one-off job present" do
    result = run_plur(
      "-C", default_ruby_dir.to_s,
      "--dry-run",
      env: {"PLUR_CONFIG_FILE" => config_file.to_s}
    )

    expect(result).to be_success
    expect(result.err).to include("[rspec]")
    expect(result.err).to include("custom-rspec")
    expect(result.err).to include("spec/calculator_spec.rb")
  end

  it "runs one-off jobs as passthrough" do
    result = run_plur(
      "-C", default_ruby_dir.to_s,
      "--dry-run",
      "--use", "lint", "spec/calculator_spec.rb",
      env: {"PLUR_CONFIG_FILE" => config_file.to_s}
    )

    expect(result).to be_success
    expect(result.err).to include("[passthrough]")
    expect(result.err).to include("echo LINT")
    expect(result.err).to include("spec/calculator_spec.rb")
    expect(result.err).not_to include("--force-color")
    expect(result.err).not_to include("--no-color")
  end

  it "uses rspec defaults for custom framework jobs without target_pattern" do
    result = run_plur(
      "-C", default_ruby_dir.to_s,
      "--no-color", "--dry-run",
      "--use", "fast",
      env: {"PLUR_CONFIG_FILE" => config_file.to_s}
    )

    expect(result).to be_success
    expect(result.err).to include("[rspec]")
    expect(result.err).to include("fast-rspec")
    expect(result.err).to include("--no-color")
    expect(result.err).to include("spec/calculator_spec.rb")
  end
end
