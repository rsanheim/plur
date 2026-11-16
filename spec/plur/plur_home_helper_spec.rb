require "spec_helper"

RSpec.describe PlurHomeHelper do
  def spawned_home
    `sh -c 'echo $HOME'`.strip
  end

  it "runs each example's subprocesses in an empty home of their own" do
    expect(spawned_home).to eq(isolated_home.to_s)
    expect(isolated_home.children).to be_empty
  end
end
