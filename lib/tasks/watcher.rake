namespace :deps do
  desc "Download watcher binary for file system monitoring"
  task :watcher do
    require "net/http"
    require "uri"
    require "fileutils"

    puts "Downloading watcher binary for platform: #{RUBY_PLATFORM}"

    # Determine platform
    platform = case RUBY_PLATFORM
    when /aarch64-darwin/, /arm64-darwin/
      "aarch64-apple-darwin"
    when /x86_64-darwin/
      raise "Intel Mac (x86_64) is not supported. Please use an Apple Silicon Mac."
    when /linux.*aarch64/, /linux.*arm64/
      "aarch64-unknown-linux-gnu"
    when /linux/
      "x86_64-unknown-linux-gnu"
    else
      raise "Unsupported platform: #{RUBY_PLATFORM}"
    end

    # Create vendor directory
    vendor_dir = File.join("rux", "vendor", "watcher")
    FileUtils.mkdir_p(vendor_dir)

    # Download URL - using v0.13.6 release
    url = "https://github.com/e-dant/watcher/releases/download/0.13.6/#{platform}.tar"

    # Download to temp file
    temp_file = File.join(vendor_dir, "watcher.tar")

    begin
      uri = URI(url)
      puts "Downloading from: #{url}"

      # Use open-uri for simpler redirect handling
      require "open-uri"

      uri.open do |remote_file|
        File.binwrite(temp_file, remote_file.read)
      end

      # Extract the binary
      puts "Extracting watcher binary..."
      Dir.chdir(vendor_dir) do
        sh "tar -xf watcher.tar"

        # The tarball contains the binary in a platform-specific directory
        binary_source = File.join(platform, "watcher")
        binary_name = "watcher-#{platform}"

        raise "Expected watcher binary not found at #{binary_source}" unless File.exist?(binary_source)

        FileUtils.mv(binary_source, binary_name)
        FileUtils.chmod(0o755, binary_name)

        # Clean up extracted directory
        FileUtils.rm_rf(platform)

        puts "Watcher binary installed at: rux/vendor/watcher/#{binary_name}"
      end
    ensure
      # Clean up temp file
      FileUtils.rm_f(temp_file)
    end
  end
end
