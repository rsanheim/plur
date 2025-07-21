require_relative "../plur"

namespace :build do
  desc "Build plur for Linux (amd64 and arm64)"
  task :linux do
    # Ensure dist directory exists
    dist_dir = Plur.config.root_dir.join("dist")
    FileUtils.mkdir_p(dist_dir)
    
    # Get version from git
    version = `git describe --tags --always --dirty`.strip
    commit = `git rev-parse HEAD`.strip
    date = Time.now.utc.strftime("%Y-%m-%d")
    
    Dir.chdir(Plur.config.plur_dir) do
      %w[amd64 arm64].each do |arch|
        puts "[build:linux] Building for linux/#{arch}..."
        
        output_file = dist_dir.join("plur-linux-#{arch}")
        
        # Set environment variables for cross-compilation
        env = {
          "GOOS" => "linux",
          "GOARCH" => arch,
          "CGO_ENABLED" => "0"  # Disable CGO for static binaries
        }
        
        # Build with ldflags to embed version info
        ldflags = [
          "-s -w",  # Strip debug info for smaller binaries
          "-X main.version=#{version}",
          "-X main.commit=#{commit}",
          "-X main.date=#{date}"
        ].join(" ")
        
        cmd = "go build -mod=mod -ldflags \"#{ldflags}\" -o #{output_file} ."
        
        # Run the build
        system(env, cmd) or raise "Build failed for linux/#{arch}"
        
        # Make it executable
        FileUtils.chmod(0755, output_file)
        
        # Show file info
        size_mb = File.size(output_file) / 1024.0 / 1024.0
        puts "[build:linux] Created #{output_file} (#{size_mb.round(1)} MB)"
      end
    end
    
    puts "[build:linux] All Linux builds completed successfully!"
    puts "[build:linux] Binaries available in: #{dist_dir}"
  end
  
  desc "Build plur for all platforms (macOS and Linux)"
  task :all do
    dist_dir = Plur.config.root_dir.join("dist")
    FileUtils.mkdir_p(dist_dir)
    
    # Get version from git
    version = `git describe --tags --always --dirty`.strip
    commit = `git rev-parse HEAD`.strip
    date = Time.now.utc.strftime("%Y-%m-%d")
    
    platforms = [
      { goos: "darwin", goarch: "amd64", name: "darwin-amd64" },
      { goos: "darwin", goarch: "arm64", name: "darwin-arm64" },
      { goos: "linux", goarch: "amd64", name: "linux-amd64" },
      { goos: "linux", goarch: "arm64", name: "linux-arm64" }
    ]
    
    Dir.chdir(Plur.config.plur_dir) do
      platforms.each do |platform|
        puts "[build:all] Building for #{platform[:name]}..."
        
        output_file = dist_dir.join("plur-#{platform[:name]}")
        
        # Set environment variables for cross-compilation
        env = {
          "GOOS" => platform[:goos],
          "GOARCH" => platform[:goarch],
          "CGO_ENABLED" => "0"  # Disable CGO for static binaries
        }
        
        # Build with ldflags to embed version info
        ldflags = [
          "-s -w",  # Strip debug info for smaller binaries
          "-X main.version=#{version}",
          "-X main.commit=#{commit}",
          "-X main.date=#{date}"
        ].join(" ")
        
        cmd = "go build -mod=mod -ldflags \"#{ldflags}\" -o #{output_file} ."
        
        # Run the build
        system(env, cmd) or raise "Build failed for #{platform[:name]}"
        
        # Make it executable
        FileUtils.chmod(0755, output_file)
        
        # Show file info
        size_mb = File.size(output_file) / 1024.0 / 1024.0
        puts "[build:all] Created #{output_file} (#{size_mb.round(1)} MB)"
      end
    end
    
    puts "[build:all] All platform builds completed successfully!"
    puts "[build:all] Binaries available in: #{dist_dir}"
  end
  
  desc "Clean multi-platform build artifacts"
  task :clean do
    dist_dir = Plur.config.root_dir.join("dist")
    if dist_dir.exist?
      FileUtils.rm_rf(dist_dir)
      puts "[build:clean] Removed dist directory"
    else
      puts "[build:clean] No dist directory to clean"
    end
  end
  
  desc "Show built binaries"
  task :list do
    dist_dir = Plur.config.root_dir.join("dist")
    
    unless dist_dir.exist?
      puts "No dist directory found. Run 'rake build:linux' or 'rake build:all' first."
      next
    end
    
    binaries = Dir[dist_dir.join("plur-*")]
    
    if binaries.empty?
      puts "No binaries found in dist directory."
      next
    end
    
    puts "Built binaries:"
    puts "-" * 60
    
    binaries.sort.each do |binary|
      size_mb = File.size(binary) / 1024.0 / 1024.0
      mtime = File.mtime(binary).strftime("%Y-%m-%d %H:%M:%S")
      name = File.basename(binary)
      
      # Try to get version from binary
      version_output = `#{binary} --version 2>/dev/null`.strip rescue "N/A"
      
      puts "  #{name}"
      puts "    Size:     #{size_mb.round(1)} MB"
      puts "    Modified: #{mtime}"
      puts "    Version:  #{version_output}" if version_output != "N/A"
      puts
    end
  end
end