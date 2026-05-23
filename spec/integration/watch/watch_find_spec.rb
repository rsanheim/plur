require "spec_helper"

RSpec.describe "plur watch find" do
  it "explains when no watch rule matches the changed file" do
    result = run_plur_allowing_errors("watch", "find", "spec/spec_helper.rb")

    expect(result.exit_status).to eq(2)
    expect(result.out).to include("[watch] Checking spec/spec_helper.rb")
    expect(result.out).to include("[watch] No matching rule for spec/spec_helper.rb")
    expect(result.out).not_to include('msg="found rules"')
  end
end
