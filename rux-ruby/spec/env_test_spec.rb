require 'spec_helper'

RSpec.describe "Environment Variables" do
  it "prints TEST_ENV_NUMBER and PARALLEL_TEST_GROUPS" do
    puts "TEST_ENV_NUMBER: '#{ENV['TEST_ENV_NUMBER']}'"
    puts "PARALLEL_TEST_GROUPS: '#{ENV['PARALLEL_TEST_GROUPS']}'"
    expect(true).to be true
  end
end