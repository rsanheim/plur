require "spec_helper"

RSpec.describe "spec_helper env scrub" do
  it "removes leaky env vars at boot, including from Bundler.with_unbundled_env" do
    env = {"NO_COLOR" => "1", "PLUR_DEV_PROFILE" => "/tmp/leak", "TEST_ENV_NUMBER" => "2", "PLUR_BINARY" => "/keep/plur"}
    probe = <<~RUBY
      keys = %w[NO_COLOR PLUR_DEV_PROFILE TEST_ENV_NUMBER PLUR_BINARY]
      p ENV.values_at(*keys)
      Bundler.with_unbundled_env { p ENV.values_at(*keys) }
    RUBY

    out, err, status = Open3.capture3(env, "bundle", "exec", "ruby", "-rrspec/core",
      "-r#{ROOT_PATH.join("spec/spec_helper")}", "-e", probe, chdir: ROOT_PATH.to_s)

    expect(status).to be_success, err
    expect(out.lines(chomp: true)).to eq([
      [nil, nil, nil, "/keep/plur"].inspect,
      [nil, nil, nil, "/keep/plur"].inspect
    ])
  end
end
