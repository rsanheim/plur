RSpec.describe "plur watch install command" do
  include PlurWatchHelper

  around_with_tmp_plur_home

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
    expect(plur_home.join("bin", watcher_filename)).to_not exist

    result = run_plur("watch", "install")

    expect(result.success?).to be true
    expect(result.out).to include("installed watcher binary path=")
    expect(result.out).to include("bin/#{watcher_filename}")

    expect(plur_home).to exist
    expect(plur_home.join("bin", watcher_filename)).to be_file
    expect(plur_home.join("bin", watcher_filename)).to be_executable
    expect(plur_home.join("bin", watcher_filename)).to exist
  end
end
