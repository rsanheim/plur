require "spec_helper"
require "json"

RSpec.describe "plur watch find/live parity" do
  include PlurWatchHelper

  it "previews the same target list that live watch executes" do
    with_temp_watch_project do |project_dir|
      find_result = chdir(project_dir) do
        run_plur("watch", "find", "--format=json", "lib/calculator.rb")
      end
      plan = JSON.parse(find_result.out)
      targets = plan.fetch("existing_targets").fetch("rspec")

      calculator_file = project_dir.join("lib/calculator.rb")
      original_content = calculator_file.read

      live_result = run_plur_watch(dir: project_dir, timeout: 10, until_output: "Executing job") do
        calculator_file.write(original_content + "\n# parity")
      end

      expect(targets).to eq(["spec/calculator_spec.rb"])
      expect(live_result.err).to include('Executing job job="rspec"')
      expect(live_result.err).to include(%(targets="[#{targets.join(" ")}]"))
    end
  end
end
