# frozen_string_literal: true

require "spec_helper"

RSpec.describe "watcher vendor platform" do
  it "maps Windows Ruby platforms to the published watcher binary" do
    code = <<~RUBY
      require "rake"
      require_relative "lib/plur"
      load "lib/tasks/vendor.rake"
      %w[x64-mingw-ucrt x64-mswin64_140].each do |platform|
        puts watcher_platform(platform)
      end
    RUBY

    output, status = Dir.chdir(ROOT_PATH) do
      Open3.capture2("bundle", "exec", "ruby", "-e", code)
    end

    expect(status).to be_success
    expect(output.lines(chomp: true)).to all(eq("x86_64-pc-windows-msvc"))
  end
end
