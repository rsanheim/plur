require "spec_helper"

RSpec.describe "plur watch run flags" do
  it "rejects explicit one-shot runner flags that do not affect live watch mode" do
    chdir(default_ruby_dir) do
      {
        ["watch", "run", "--workers=99", "--timeout", "1"] => "--workers",
        ["watch", "run", "-n", "2", "--timeout", "1"] => "--workers",
        ["watch", "run", "--dry-run-format=json", "--timeout", "1"] => "--dry-run-format",
        ["watch", "run", "--first-is-1", "--timeout", "1"] => "--first-is-1",
        ["watch", "run", "--no-first-is-1", "--timeout", "1"] => "--no-first-is-1"
      }.each do |args, flag|
        result = run_plur_allowing_errors(*args)

        expect(result.exit_status).to eq(1)
        expect(result.out).to eq("")
        expect(result.err).to include("#{flag} does not apply to plur watch run")
        expect(result.err).not_to include("ready and watching")
      end
    end
  end
end
