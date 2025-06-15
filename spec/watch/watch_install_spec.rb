RSpec.describe "rux watch install command" do
  include RuxWatchHelper

  around_with_tmp_rux_home

  def watcher_filename
    case RUBY_PLATFORM
    when /aarch64-darwin/, /arm64-darwin/
      "watcher-aarch64-apple-darwin"
    when /linux.*aarch64/, /linux.*arm64/
      "watcher-aarch64-unknown-linux-gnu"
    when /linux/
      "watcher-x86_64-unknown-linux-gnu"
    else
      raise "Unsupported platform: #{RUBY_PLATFORM}"
    end
  end

  it "installs the watcher binary in RUX_HOME/bin/[platform-specific-binary]" do
    expect(rux_home.join("bin", watcher_filename)).to_not exist

    result = run_rux("watch", "install")

    expect(result.success?).to be true
    expect(result.out).to include("installed watcher binary path=")
    expect(result.out).to include("bin/#{watcher_filename}")

    expect(rux_home).to exist
    expect(rux_home.join("bin", watcher_filename)).to be_file
    expect(rux_home.join("bin", watcher_filename)).to be_executable
    expect(rux_home.join("bin", watcher_filename)).to exist
  end
end
