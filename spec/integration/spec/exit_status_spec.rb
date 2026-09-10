require "spec_helper"

RSpec.describe "Plur RSpec exit status" do
  around_with_tmp_plur_home

  around do |example|
    FileUtils.mkdir_p(ROOT_PATH.join("tmp"))
    Dir.mktmpdir("rspec-exit-status-", ROOT_PATH.join("tmp")) do |project|
      FileUtils.mkdir_p(File.join(project, "spec"))
      File.write(File.join(project, ".rspec"), "--require ./spec/spec_helper.rb\n")
      File.write(File.join(project, "spec/spec_helper.rb"), <<~RUBY)
        RSpec.configure do |config|
          config.order = :defined
          config.failure_exit_code = Integer(ENV.fetch("SPEC_FAILURE_CODE", "1"))
          config.error_exit_code = Integer(ENV["SPEC_ERROR_CODE"]) if ENV["SPEC_ERROR_CODE"]
        end
      RUBY
      File.write(File.join(project, "spec/passing_spec.rb"), <<~RUBY)
        RSpec.describe "passing" do
          it("passes") { expect(1).to eq(1) }
        end
      RUBY
      File.write(File.join(project, "spec/failing_spec.rb"), <<~RUBY)
        RSpec.describe "failing" do
          it("fails") { expect(1).to eq(2) }
        end
      RUBY
      File.write(File.join(project, "spec/load_error_spec.rb"), 'raise "load error"')
      File.write(File.join(project, "spec/suite_error_spec.rb"), <<~RUBY)
        RSpec.configure { |config| config.after(:suite) { raise "suite error" } }
        RSpec.describe "suite error" do
          it("passes before the suite hook fails") { expect(1).to eq(1) }
        end
      RUBY
      File.write(File.join(project, "spec/exit_spec.rb"), <<~RUBY)
        RSpec.describe "explicit exit" do
          it("passes before exiting") { expect(1).to eq(1) }
          it("exits") { exit! Integer(ENV.fetch("SPEC_EXIT_CODE", "42")) }
        end
      RUBY
      File.write(File.join(project, "spec/signal_spec.rb"), <<~RUBY)
        RSpec.describe "terminated worker" do
          it("passes before termination") { expect(1).to eq(1) }
          it("terminates") { Process.kill("TERM", Process.pid) }
        end
      RUBY
      File.write(File.join(project, "spec/after_exit_spec.rb"), <<~RUBY)
        at_exit { exit 42 }
        RSpec.describe "exit after completion" do
          it("passes") { expect(1).to eq(1) }
        end
      RUBY
      File.write(File.join(project, "spec/after_signal_spec.rb"), <<~RUBY)
        at_exit { Process.kill("TERM", Process.pid) }
        RSpec.describe "termination after completion report" do
          it("passes") { expect(1).to eq(1) }
        end
      RUBY

      Dir.chdir(project) { example.run }
    end
  end

  [
    {name: "passing examples", files: %w[passing], code: 0},
    {name: "a completed run with all examples filtered out", files: %w[passing failing], args: %w[--tag absent], code: 0},
    {name: "assertion failures", files: %w[passing failing], code: 1},
    {name: "load errors", files: %w[passing load_error], code: 1},
    {name: "configured failure codes", files: %w[passing failing], env: {"SPEC_FAILURE_CODE" => "17"}, code: 17},
    {name: "a configured failure code of 2", files: %w[passing failing], env: {"SPEC_FAILURE_CODE" => "2"}, code: 2},
    {name: "a configured failure code of 70", files: %w[passing failing], env: {"SPEC_FAILURE_CODE" => "70"}, code: 70},
    {name: "CLI failure codes", files: %w[passing failing], args: %w[--failure-exit-code 19], code: 19},
    {name: "load errors using the failure code", files: %w[passing load_error], env: {"SPEC_FAILURE_CODE" => "17"}, code: 17},
    {name: "configured error codes", files: %w[passing load_error], env: {"SPEC_FAILURE_CODE" => "17", "SPEC_ERROR_CODE" => "3"}, code: 3},
    {name: "suite errors taking precedence over assertion failures", files: %w[failing suite_error], env: {"SPEC_FAILURE_CODE" => "17", "SPEC_ERROR_CODE" => "3"}, code: 3},
    {name: "an exit code set after RSpec completion", files: %w[passing after_exit], code: 42}
  ].each do |scenario|
    it "matches RSpec for #{scenario.fetch(:name)}" do
      files = scenario.fetch(:files).map { |name| "spec/#{name}_spec.rb" }
      env = scenario.fetch(:env, {})
      args = scenario.fetch(:args, [])
      rspec_out, rspec_err, rspec_status = Open3.capture3(env,
        "bundle", "exec", "rspec", *files, *args, "--no-color")
      rspec_code = rspec_status.exitstatus || (128 + rspec_status.termsig)
      expect(rspec_code).to eq(scenario.fetch(:code)), "RSpec exited #{rspec_code}:\n#{rspec_out}\n#{rspec_err}"

      [1, 2].each do |workers|
        FileUtils.rm_rf(plur_home.join("runtime"))
        plur_args = ["-n", workers.to_s, "--color=never", *files]
        plur_args.concat(["--", *args]) unless args.empty?
        result = run_plur_allowing_errors(*plur_args, env: env)
        expect(result.exit_status).to eq(rspec_code), "With #{workers} workers: expected RSpec status #{rspec_code}, got Plur status #{result.exit_status}:\n#{result.out}\n#{result.err}"
        expect(result.err).not_to include("terminated abnormally", "without an RSpec completion report")
      end
    end
  end

  [
    {name: "exit! 0 before completion", files: %w[passing exit], env: {"SPEC_EXIT_CODE" => "0"}, code: 0},
    {name: "exit! 1 before completion", files: %w[passing exit], env: {"SPEC_EXIT_CODE" => "1"}, code: 1},
    {name: "exit! 42 before completion", files: %w[passing exit], code: 42},
    {name: "signal termination", files: %w[passing signal], code: 143},
    {name: "signal termination after the completion report", files: %w[passing after_signal], code: 143},
    {name: "an abnormal exit alongside assertion failures", files: %w[failing exit], env: {"SPEC_FAILURE_CODE" => "17"}, code: 42}
  ].each do |scenario|
    it "returns Plur's worker error code for #{scenario.fetch(:name)}" do
      files = scenario.fetch(:files).map { |name| "spec/#{name}_spec.rb" }
      env = scenario.fetch(:env, {})
      _out, _err, rspec_status = Open3.capture3(env, "bundle", "exec", "rspec", *files, "--no-color")
      rspec_code = rspec_status.exitstatus || (128 + rspec_status.termsig)
      expect(rspec_code).to eq(scenario.fetch(:code))

      [1, 2].each do |workers|
        FileUtils.rm_rf(plur_home.join("runtime"))
        result = run_plur_allowing_errors("-n", workers.to_s, "--color=never", *files, env: env)
        expect(result.exit_status).to eq(70), "With #{workers} workers:\n#{result.out}\n#{result.err}"
        expect(result.err).to match(/worker \d+ (terminated abnormally|exited without an RSpec completion report)/)
      end
    end
  end
end
