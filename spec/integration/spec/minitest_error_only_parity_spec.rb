require "spec_helper"

# Errors are not failures. A run whose tests all raise must report minitest's
# own counts - N errors, 0 failures, 0 assertions - not per-test tallies,
# which see every non-passing test as a failure. Minitest run directly on the
# same file is the oracle.
RSpec.describe "Minitest error-only parity" do
  before(:all) do
    chdir(project_fixture!("minitest-outcomes")) do
      Bundler.with_unbundled_env do
        system("bundle check", out: File::NULL, err: File::NULL) ||
          system("bundle install", out: File::NULL, err: File::NULL, exception: true)
      end
    end
  end

  # Order- and runner-independent facts: outcome counts and which tests ran.
  def outcome_facts(snapshot)
    stdout = snapshot.fetch("stdout", "")
    counts = stdout.scan(/(\d+) runs?, (\d+) assertions?, (\d+) failures?, (\d+) errors?, (\d+) skips?/).last
    ids = stdout.scan(/[A-Z]\w*Test#test_\w+/).uniq.sort
    snapshot.merge("stdout" => "counts=#{counts&.join(",")}\nids=#{ids.join(",")}", "stderr" => "")
  end

  it "reports the same counts as minitest when every test errors" do
    chdir(project_fixture!("minitest-outcomes")) do
      Bundler.with_unbundled_env do
        error_file = "test/errors_only_test.rb"
        File.write(error_file, <<~RUBY)
          require "minitest/autorun"

          class ErrorsOnlyTest < Minitest::Test
            def test_raises
              raise "TOKEN_ERR_ONE"
            end

            def test_also_raises
              raise "TOKEN_ERR_TWO"
            end
          end
        RUBY

        result = Backspin.compare(
          reference: ["bundle", "exec", "ruby", "-Itest", "-e", %(require "errors_only_test")],
          actual: [plur_binary, "--use", "minitest", "-n", "1", "--color=never", error_file],
          filter: ->(snapshot) { outcome_facts(snapshot) }
        )

        expect(result.verified?).to be true
        expect(result.actual.status).to eq(1)
      ensure
        FileUtils.rm_f(error_file)
      end
    end
  end
end
