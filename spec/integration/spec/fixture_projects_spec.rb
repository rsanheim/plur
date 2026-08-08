require "spec_helper"
require_relative "../../support/fixture_runner"

RSpec.describe "Fixture Projects" do
  # Minitest fixtures are validated in minitest_integration_spec.rb, whose
  # Backspin comparison already raw-runs minitest-outcomes against plur.
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
