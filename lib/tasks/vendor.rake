require "pathname"

WATCHER_VERSION = "0.13.6"

# Helper method to determine platform
def watcher_platform
  case RUBY_PLATFORM
  when /aarch64-darwin/, /arm64-darwin/
    "aarch64-apple-darwin"
  when /linux.*aarch64/, /linux.*arm64/
    "aarch64-unknown-linux-gnu"
  when /linux/
    "x86_64-unknown-linux-gnu"
  else
    raise "Unsupported platform: #{RUBY_PLATFORM}"
  end
end

def watcher_binary_name
  "watcher-#{watcher_platform}"
end

def watcher_binary_path
  Pathname.new("rux/vendor/watcher").join(watcher_binary_name).to_s
end

# Directory for watcher binaries
directory "rux/vendor/watcher"

# File task for the watcher binary
file watcher_binary_path => ["rux/vendor/watcher"] do
  require "net/http"
  require "uri/https"
  require "open-uri"
  require "tmpdir"

  puts "Downloading watcher binary for platform: #{RUBY_PLATFORM}"

  platform = watcher_platform
  vendor_dir = Pathname.new("rux/vendor/watcher").expand_path

  # Download URL
  url = "https://github.com/e-dant/watcher/releases/download/#{WATCHER_VERSION}/#{platform}.tar"

  # Use a temporary directory for download and extraction
  Dir.mktmpdir do |tmpdir|
    tmpdir_path = Pathname.new(tmpdir)
    temp_file = tmpdir_path.join("watcher.tar")

    uri = URI(url)
    puts "Downloading from: #{url}"

    uri.open do |remote_file|
      temp_file.binwrite(remote_file.read)
    end

    # Extract the binary
    puts "Extracting watcher binary..."
    sh "tar -xf #{temp_file} -C #{tmpdir}"

    # The tarball contains the binary in a platform-specific directory
    binary_source = tmpdir_path.join(platform, "watcher")
    binary_name = watcher_binary_name
    final_path = vendor_dir.join(binary_name)

    raise "Expected watcher binary not found at #{binary_source}" unless binary_source.exist?

    # Move to final destination
    FileUtils.mv(binary_source.to_s, final_path.to_s)
    final_path.chmod(0o755)

    puts "Watcher binary installed at: #{watcher_binary_path}"
  end
end

namespace :vendor do
  desc "Build vendored dependencies"
  task build: [watcher_binary_path]

  desc "Clean vendored dependencies"
  task :clean do
    vendor_dir = Pathname.new("rux/vendor/watcher")
    if vendor_dir.exist?
      vendor_dir.rmtree
      puts "Removed vendored dependencies from #{vendor_dir}"
    else
      puts "No vendored dependencies to clean"
    end
  end
end
