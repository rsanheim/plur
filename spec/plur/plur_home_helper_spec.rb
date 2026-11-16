require "spec_helper"

RSpec.describe PlurHomeHelper do
  def spawned_home
    `sh -c 'echo $HOME'`.strip
  end

  it "runs each example's subprocesses in an empty home of their own" do
    expect(spawned_home).to eq(isolated_home.to_s)
    expect(isolated_home.children).to be_empty
  end

  it "keeps that home inside with_unbundled_env" do
    with_unbundled_env do
      expect(ENV).not_to include("BUNDLE_GEMFILE")
      expect(spawned_home).to eq(isolated_home.to_s)
    end
  end

  it "is the only place that calls Bundler's own with_unbundled_env, which restores the real HOME" do
    callers = Dir.glob("spec/**/*.rb", base: ROOT_PATH).select do |file|
      ROOT_PATH.join(file).read.match?(/Bundler\.with_unbundled_env/)
    end

    expect(callers).to eq(["spec/support/plur_home_helper.rb"])
  end
end
