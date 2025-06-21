require 'spec_helper'
require_relative 'support/fixture_runner'

RSpec.describe "Fixture Projects" do
  describe "minitest projects" do
    it "runs minitest-success project with all tests passing" do
      result = run_fixture_tests("minitest-success", framework: :minitest)
      
      expect(result).to be_success
      expect(result.exit_status).to eq 0
      expect(result.out).to include("0 failures")
      expect(result.out).to include("0 errors")
      expect(result.out).to include("in test_addition")
      expect(result.out).to include("in test_titleize")
    end

    it "runs minitest-failures project with expected failures" do
      result = run_fixture_tests("minitest-failures", framework: :minitest)
      
      expect(result).to be_failure
      expect(result.exit_status).to_not eq 0
      expect(result.out).to include("failures")
      
      # Check for specific expected failures
      expect(result.out).to include("test_display_name_failure")
      expect(result.out).to include("test_email_validation_mixed")
    end
  end

  describe "test-unit projects" do
    it "runs testunit-success project with all tests passing" do
      result = run_fixture_tests("testunit-success", framework: :testunit)
      
      expect(result).to be_success
      expect(result.exit_status).to eq 0
      expect(result.out).to include("0 failures")
      expect(result.out).to include("0 errors")
    end

    it "runs testunit-failures project with expected failures" do
      result = run_fixture_tests("testunit-failures", framework: :testunit)
      
      expect(result).to be_failure
      expect(result.exit_status).to_not eq 0
      expect(result.out).to include("Failure")
      
      # Check for specific expected failures
      expect(result.out).to include("test_withdraw_insufficient_funds_wrong_error")
      expect(result.out).to include("test_balance_after_multiple_operations_failure")
    end
  end
end