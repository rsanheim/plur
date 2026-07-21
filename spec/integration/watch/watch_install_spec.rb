RSpec.describe "plur watch install command" do
  include PlurWatchHelper

  around_with_tmp_plur_home

  def watcher_filename
    case RUBY_PLATFORM
    when /aarch64-darwin/, /arm64-darwin/
      "watcher-aarch64-apple-darwin"
    when /linux.*aarch64/, /linux.*arm64/, /aarch64.*linux/
      "watcher-aarch64-unknown-linux-gnu"
    when /linux/
      "watcher-x86_64-unknown-linux-gnu"
    else
      raise "Unsupported platform: #{RUBY_PLATFORM}"
    end
  end

  it "installs the watcher binary for #{RUBY_PLATFORM} in PLUR_HOME/bin/[platform-specific-binary]" do
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

  it "leaves an existing binary untouched without --force" do
    binary = plur_home.join("bin", watcher_filename)
    binary.dirname.mkpath
    binary.write("sentinel")

    result = run_plur("watch", "install")

    expect(result.success?).to be true
    expect(binary.read).to eq("sentinel")
  end

  it "reinstalls over an existing binary with --force" do
    binary = plur_home.join("bin", watcher_filename)
    binary.dirname.mkpath
    binary.write("sentinel")
    binary.chmod(0o755)

    result = run_plur("watch", "install", "--force")

    expect(result.success?).to be true
    expect(result.out).to include("installed watcher binary path=")
    expect(binary.read).to_not eq("sentinel")
    expect(binary).to be_executable
  end
end
