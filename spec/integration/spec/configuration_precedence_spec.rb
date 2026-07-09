require "spec_helper"

# Worker-count precedence is a documented contract, so this file pins every rung
# of the ladder. Highest precedence wins:
#
#   CLI flag  >  PLUR_WORKERS  >  PARALLEL_TEST_PROCESSORS  >  config file  >  built-in default
#
# Each example drives the real `plur` binary and reads the resolved value back
# from `plur doctor`, which reports the fully-resolved worker count. We assert on
# doctor (not `--dry-run` worker listings) on purpose: dry-run caps the visible
# worker count at the number of test files, so a fixture with two spec files can
# never distinguish "2 requested" from "16 requested, capped to 2" — the exact
# blind spot that let the earlier precedence specs pass while asserting the wrong
# direction.
#
# Every run is hermetic: a fresh project dir (no stray .plur.toml), a fresh
# HOME/PLUR_HOME (no user ~/.plur.toml), NO_COLOR so the number parses cleanly,
# and the worker env vars stripped from the parent process for the duration of
# each example so a suite running under plur/parallel_tests cannot taint the child.
RSpec.describe "Configuration precedence (worker count)" do
  # Env vars that determine, or could leak into, worker-count resolution. They are
  # stripped from the parent for each example; every example re-adds only the ones
  # it is exercising.
  around do |example|
    keys = %w[PLUR_WORKERS PARALLEL_TEST_PROCESSORS PARALLEL_TEST_GROUPS TEST_ENV_NUMBER PLUR_CONFIG_FILE PLUR_DEBUG]
    saved = keys.to_h { |key| [key, ENV[key]] }
    keys.each { |key| ENV.delete(key) }
    begin
      example.run
    ensure
      saved.each do |key, value|
        value.nil? ? ENV.delete(key) : ENV[key] = value
      end
    end
  end

  it "uses the built-in default of 4 when nothing is set" do
    expect(resolved_workers).to eq(4)
  end

  it "uses the config file value over the built-in default" do
    expect(resolved_workers(config_workers: 2)).to eq(2)
  end

  it "lets PARALLEL_TEST_PROCESSORS override the config file" do
    expect(resolved_workers(config_workers: 2, env: {"PARALLEL_TEST_PROCESSORS" => "16"})).to eq(16)
  end

  it "lets PLUR_WORKERS override the config file" do
    expect(resolved_workers(config_workers: 2, env: {"PLUR_WORKERS" => "7"})).to eq(7)
  end

  it "prefers PLUR_WORKERS over PARALLEL_TEST_PROCESSORS when both are set" do
    expect(
      resolved_workers(config_workers: 2, env: {"PLUR_WORKERS" => "7", "PARALLEL_TEST_PROCESSORS" => "16"})
    ).to eq(7)
  end

  it "lets PLUR_WORKERS override the built-in default with no config file present" do
    expect(resolved_workers(env: {"PLUR_WORKERS" => "7"})).to eq(7)
  end

  it "lets PARALLEL_TEST_PROCESSORS override the built-in default with no config file present" do
    expect(resolved_workers(env: {"PARALLEL_TEST_PROCESSORS" => "16"})).to eq(16)
  end

  it "lets the --workers flag override every env var and the config file" do
    expect(
      resolved_workers(
        config_workers: 2,
        env: {"PLUR_WORKERS" => "7", "PARALLEL_TEST_PROCESSORS" => "16"},
        args: ["--workers=3"]
      )
    ).to eq(3)
  end

  # Runs `plur doctor` in a hermetic sandbox and returns the resolved worker count
  # it reports. doctor computes that count through the same resolver path as a real
  # run, which is what makes these assertions discriminate — keep it on doctor, not
  # dry-run. config_workers, when given, is written to a project .plur.toml; env
  # adds child environment variables; args are extra CLI arguments.
  def resolved_workers(config_workers: nil, env: {}, args: [])
    Dir.mktmpdir do |project|
      Dir.mktmpdir do |home|
        if config_workers
          File.write(File.join(project, ".plur.toml"), "workers = #{config_workers}\n")
        end

        base_env = {
          "HOME" => home,
          "PLUR_HOME" => File.join(home, ".plur"),
          "NO_COLOR" => "1"
        }
        output, error, status = Open3.capture3(
          base_env.merge(env),
          plur_binary, "doctor", *args,
          chdir: project
        )

        expect(status).to be_success, "plur doctor failed (#{status.exitstatus}):\n#{error}"
        # Anchored + title-case on purpose: `Workers:` must not match doctor's
        # raw `  PLUR_WORKERS:` echo line, only the resolved `  Workers:` line.
        workers = output[/^\s*Workers:\s+(\d+)/, 1]
        expect(workers).not_to be_nil, "no 'Workers:' line in doctor output:\n#{output}"
        Integer(workers)
      end
    end
  end
end
