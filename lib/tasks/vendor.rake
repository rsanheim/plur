# require_relative "../plur"

# Helper method to determine platform
def watcher_platform
  case RUBY_PLATFORM
  when /aarch64-darwin/, /arm64-darwin/
    "aarch64-apple-darwin"
  when /linux.*aarch64/, /linux.*arm64/, /aarch64.*linux/
    "aarch64-unknown-linux-gnu"
  when /linux/
    "x86_64-unknown-linux-gnu"
  else
    raise "Unsupported platform: #{RUBY_PLATFORM}"
  end
end

# All platforms we support
def all_watcher_platforms
  [
    "aarch64-apple-darwin",       # macOS ARM64
    "aarch64-unknown-linux-gnu",  # Linux ARM64
    "x86_64-unknown-linux-gnu",   # Linux x64
    "x86_64-pc-windows-msvc"      # Windows x64 (experimental)
  ]
end

def watcher_binary_name(platform = nil)
  platform ||= watcher_platform
  "watcher-#{platform}"
end

def watcher_binary_path(platform = nil)
  Plur.config.watcher_dir.join(watcher_binary_name(platform)).to_s
end

# Directory for watcher binaries
directory Plur.config.watcher_dir.to_s

# Download a specific watcher binary
def download_watcher_binary(platform)
  require "net/http"
  require "uri/https"
  require "open-uri"
  require "tmpdir"
  require "digest"

  binary_path = watcher_binary_path(platform)

  # Skip if already exists
  if File.exist?(binary_path)
    puts "Watcher binary already exists for #{platform}"
    return
  end

  puts "Downloading watcher binary for platform: #{platform}"

  # Download URL
  url = "https://github.com/e-dant/watcher/releases/download/#{Plur.config.edant_watcher_version}/#{platform}.tar"

  # Use a temporary directory for download and extraction
  Dir.mktmpdir do |tmpdir|
    tmpdir_path = Pathname.new(tmpdir)
    temp_file = tmpdir_path.join("watcher.tar")

    uri = URI(url)
    puts "Downloading from: #{url}"

    uri.open do |remote_file|
      temp_file.binwrite(remote_file.read)
    end

    # Verify checksum
    checksum_url = "#{url}.sha256sum"
    expected_checksum = URI(checksum_url).open { |f| f.read.strip }
    actual_checksum = Digest::SHA256.file(temp_file).hexdigest

    if actual_checksum != expected_checksum
      raise "Checksum mismatch for #{platform}.tar!\n" \
            "  Expected: #{expected_checksum}\n" \
            "  Actual:   #{actual_checksum}"
    end
    puts "Checksum verified: #{actual_checksum[0..7]}..."

    # Extract the binary
    puts "Extracting watcher binary..."
    sh "tar -xf #{temp_file} -C #{tmpdir}"

    # The tarball contains the binary in a platform-specific directory
    binary_source = tmpdir_path.join(platform, "watcher")
    binary_name = watcher_binary_name(platform)
    final_path = Plur.config.watcher_dir.join(binary_name)

    raise "Expected watcher binary not found at #{binary_source}" unless binary_source.exist?

    # Move to final destination
    FileUtils.mv(binary_source.to_s, final_path.to_s)
    final_path.chmod(0o755)

    puts "Watcher binary installed at: #{final_path}"
  end
end

# File task for the current platform watcher binary
file watcher_binary_path => [Plur.config.watcher_dir.to_s] do
  download_watcher_binary(watcher_platform)
end

namespace :vendor do
  namespace :download do
    desc "Download vendored dependencies for current platform"
    task current: [watcher_binary_path]

    desc "Download vendored dependencies for all supported platforms"
    task all: [Plur.config.watcher_dir.to_s] do
      all_watcher_platforms.each do |platform|
        download_watcher_binary(platform)
      end
      puts "All platform watcher binaries downloaded successfully!"
    end
  end

  desc "Download vendored dependencies for current platform"
  task download: ["download:current"]

  desc "Clean vendored dependencies"
  task :clean do
    if Plur.config.watcher_dir.exist?
      # Remove all platform binaries
      all_watcher_platforms.each do |platform|
        binary_path = watcher_binary_path(platform)
        if File.exist?(binary_path)
          File.delete(binary_path)
          puts "Removed #{watcher_binary_name(platform)}"
        end
      end

      # Also remove any other files in the directory
      Dir.glob(Plur.config.watcher_dir.join("*")).each do |file|
        File.delete(file) if File.file?(file)
      end

      # Remove the directory if empty
      Plur.config.watcher_dir.rmdir if Plur.config.watcher_dir.empty?
      puts "Cleaned vendored dependencies from #{Plur.config.watcher_dir}"
    else
      puts "No vendored dependencies to clean"
    end
  end
end
