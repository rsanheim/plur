require "spec_helper"
require "json"

RSpec.describe "plur watch find JSON output" do
  it "prints structured runnable target details" do
    chdir(default_ruby_dir) do
      result = run_plur("watch", "find", "--format=json", "lib/calculator.rb")
      plan = JSON.parse(result.out)

      expect(plan).to include(
        "version" => 1,
        "mode" => "watch_find",
        "file" => "lib/calculator.rb",
        "exit_code" => 0
      )
      expect(plan["matched_rules"].first).to include(
        "name" => "lib-to-spec",
        "source" => "lib/**/*.rb",
        "jobs" => ["rspec"],
        "target" => "spec/{{match}}_spec.rb"
      )
      expect(plan["existing_targets"]).to include("rspec" => ["spec/calculator_spec.rb"])
      expect(plan["missing_targets"]).to eq({})
    end
  end

  it "prints structured no-rule details with exit 2" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors("watch", "find", "--format=json", "spec/spec_helper.rb")
      plan = JSON.parse(result.out)

      expect(result.exit_status).to eq(2)
      expect(plan).to include(
        "version" => 1,
        "mode" => "watch_find",
        "file" => "spec/spec_helper.rb",
        "exit_code" => 2
      )
      expect(plan["matched_rules"]).to eq([])
      expect(plan["existing_targets"]).to eq({})
      expect(plan["missing_targets"]).to eq({})
    end
  end
end
