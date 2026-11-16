require "spec_helper"

RSpec.describe PlurHomeHelper do
  def spawned_home
    `sh -c 'echo $HOME'`.strip
  end

  it "runs each example's subprocesses in an empty home of their own" do
    expect(spawned_home).to eq(isolated_home.to_s)
    expect(isolated_home.children).to be_empty
  end

  it "is the only caller of Bundler's with_unbundled_env, which restores the real HOME" do
    callers = `git -C #{ROOT_PATH} grep -l 'Bundler\.with_unbundled_env' -- spec`.split("\n")

    expect(callers).to eq(["spec/support/plur_home_helper.rb"])
  end
end
