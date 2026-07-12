require "spec_helper"

# Color resolution is a documented contract, so this file pins every rung of the
# ladder as seen through a pipe (which is how agents and CI always run plur):
#
#   --color=always|never  >  FORCE_COLOR / CLICOLOR_FORCE / NO_COLOR  >  config file  >  auto (tty detection)
#
# run_plur drives the real binary through pipes, so "no flags, clean env" means
# auto resolves to no color. TTY-side behavior lives in tty_output_spec.rb.
RSpec.describe "Color resolution over a pipe" do
  def ansi
    /\e\[\d+m/
  end

  def run_mixed(*args, env: {})
    chdir(project_fixture("failing_specs")) do
      run_plur_allowing_errors(*args, "spec/mixed_results_spec.rb", env: env)
    end
  end

  context "auto (default)" do
    it "emits no ANSI codes on a pipe" do
      result = run_mixed
      expect(result.out).not_to match(ansi)
      expect(result.out).to include("F")
      expect(result.out).to include(".")
    end
  end

  context "explicit flag" do
    it "--color=always emits ANSI on a pipe" do
      result = run_mixed("--color=always")
      expect(result.out).to include("\e[31mF\e[0m")
      expect(result.out).to include("\e[32m.\e[0m")
    end

    it "--color=on is an alias for always" do
      result = run_mixed("--color=on")
      expect(result.out).to match(ansi)
    end

    it "--color=never emits no ANSI" do
      result = run_mixed("--color=never")
      expect(result.out).not_to match(ansi)
    end

    it "--color=always beats NO_COLOR" do
      result = run_mixed("--color=always", env: {"NO_COLOR" => "1"})
      expect(result.out).to match(ansi)
    end
  end

  context "environment conventions" do
    it "FORCE_COLOR=1 forces ANSI on a pipe" do
      result = run_mixed(env: {"FORCE_COLOR" => "1"})
      expect(result.out).to match(ansi)
    end

    it "CLICOLOR_FORCE=1 forces ANSI on a pipe" do
      result = run_mixed(env: {"CLICOLOR_FORCE" => "1"})
      expect(result.out).to match(ansi)
    end

    it "FORCE_COLOR=0 does not force" do
      result = run_mixed(env: {"FORCE_COLOR" => "0"})
      expect(result.out).not_to match(ansi)
    end

    it "NO_COLOR disables color" do
      result = run_mixed(env: {"NO_COLOR" => "1"})
      expect(result.out).not_to match(ansi)
    end

    it "FORCE_COLOR beats NO_COLOR when both are set" do
      result = run_mixed(env: {"FORCE_COLOR" => "1", "NO_COLOR" => "1"})
      expect(result.out).to match(ansi)
    end
  end

  context "config file" do
    def with_color_config(value)
      Dir.mktmpdir do |dir|
        config_path = File.join(dir, "color.toml")
        File.write(config_path, %(color = #{value}\n))
        yield config_path
      end
    end

    it "color = \"always\" emits ANSI on a pipe" do
      with_color_config(%("always")) do |config_path|
        result = run_mixed(env: {"PLUR_CONFIG_FILE" => config_path})
        expect(result.out).to match(ansi)
      end
    end

    it "env beats config: color = \"never\" loses to FORCE_COLOR=1" do
      with_color_config(%("never")) do |config_path|
        result = run_mixed(env: {"PLUR_CONFIG_FILE" => config_path, "FORCE_COLOR" => "1"})
        expect(result.out).to match(ansi)
      end
    end

    it "rejects the retired boolean form with a useful error" do
      with_color_config("true") do |config_path|
        result = run_mixed(env: {"PLUR_CONFIG_FILE" => config_path})
        expect(result.exit_status).not_to eq(0)
        expect(result.err).to include('use "auto", "always", or "never"')
      end
    end
  end

  context "retired flag forms" do
    it "bare --color errors" do
      result = run_mixed("--color")
      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("--color")
    end

    it "--no-color errors" do
      result = run_mixed("--no-color")
      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("unknown flag")
    end
  end
end
