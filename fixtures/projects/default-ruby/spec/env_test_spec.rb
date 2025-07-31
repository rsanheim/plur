require "spec_helper"

# rubocop:disable RuboCop::Cop::Style::StderrPuts 
RSpec.describe "Environment Variables" do
  it "prints TEST_ENV_NUMBER and PARALLEL_TEST_GROUPS" do
    warn "TEST_ENV_NUMBER: '#{ENV["TEST_ENV_NUMBER"]}'"
    warn "PARALLEL_TEST_GROUPS: '#{ENV["PARALLEL_TEST_GROUPS"]}'"
    expect(true).to be true
  end
end
# rubocop:enable RuboCop::Cop::Style::StderrPuts 