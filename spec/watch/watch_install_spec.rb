RSpec.describe "rux watch install command" do
  include RuxWatchHelper

  around do |example|
    Dir.mktmpdir do |tmpdir|
      rux_home = Pathname.new(tmpdir)
      ENV["RUX_HOME"] = rux_home.to_s
      example.run
    ensure
      ENV.delete("RUX_HOME")
    end
  end

  def rux_home
    Pathname.new(ENV["RUX_HOME"])
  end

  it "installs the watcher binary in RUX_HOME/bin/[platform-specific-binary]" do
    expect(rux_home.join("bin", "watcher-aarch64-apple-darwin")).to_not exist

    result = run_rux("watch", "install")

    expect(result.success?).to be true
    expect(result.out).to include("installed watcher binary path=")
    expect(result.out).to include("bin/watcher-aarch64-apple-darwin")

    expect(rux_home).to exist
    expect(rux_home.join("bin", "watcher-aarch64-apple-darwin")).to be_file
    expect(rux_home.join("bin", "watcher-aarch64-apple-darwin")).to be_executable
    expect(rux_home.join("bin", "watcher-aarch64-apple-darwin")).to exist
  end
end
