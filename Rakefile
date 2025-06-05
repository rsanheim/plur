require "rake"
require "fileutils"

begin
  require "bundler"
  require "standard/rake" if Gem::Specification.find_all_by_name("standard").any?
  require "rspec/core/rake_task" if Gem::Specification.find_all_by_name("rspec").any?
rescue LoadError
end

# Define RSpec task if available
RSpec::Core::RakeTask.new(:spec) if defined?(RSpec)

# Default task runs all checks
desc "Run all tests and linting"
task default: ["test:all", "lint:all"]

task install: [:build_release] do
  Dir.chdir("rux") do
    # Copy the built binary to GOPATH/bin
    sh "cp rux $(go env GOPATH)/bin/"
  end
end

# ========================================
# Test Tasks
# ========================================
# For Go tests:
#   - Default: minimal output (just pass/fail)
#   - VERBOSE=1 rake test:go: detailed test output
#   - Install gotestsum for better formatting:
#     go install gotest.tools/gotestsum@latest
# ========================================
namespace :test do
  desc "Run all tests (Go, Ruby, and Integration)"
  task all: %i[go ruby integration]

  desc "Run Go tests"
  task :go do
    Dir.chdir("rux") do
      puts "Running Go tests..."

      # Download dependencies first
      puts "Downloading Go dependencies..."
      sh "go mod download"

      # Clean up tmp directory first to avoid test artifacts
      if Dir.exist?("tmp")
        puts "Cleaning up tmp directory..."
        FileUtils.rm_rf("tmp")
        FileUtils.mkdir_p("tmp")
      end

      # Ensure we're in the right directory and can find embedded files
      raise "Cannot find rspec/formatter.rb - are we in the right directory?" unless File.exist?("rspec/formatter.rb")

      # Use standard output format (dots) unless verbose is requested
      if ENV["VERBOSE"]
        sh "go test -v ./..."
      elsif system("which gotestsum > /dev/null 2>&1")
        # Use gotestsum for better formatting if available
        sh "gotestsum --format short ./..."
      else
        # This gives a cleaner output with just pass/fail
        sh "go test ./..."
      end
    end
  end

  desc "Run Ruby tests with rux"
  task ruby: %i[build rux_ruby]

  desc "Run rux-ruby specs using rux (excluding failing examples)"
  task :rux_ruby do
    Dir.chdir("rux-ruby") do
      puts "Running rux-ruby specs with rux (excluding failing_examples_spec.rb)..."
      spec_files = Dir.glob("spec/**/*_spec.rb").reject { |f| f.include?("failing_examples_spec.rb") }

      # Use rux from PATH if available, otherwise use relative path
      rux_command = system("which rux > /dev/null 2>&1") ? "rux" : "../rux/rux"

      sh "#{rux_command} #{spec_files.join(" ")}"
    end
  end

  desc "Run rux-ruby specs using turbo_tests (excluding failing examples)"
  task :rux_ruby_turbo do
    Dir.chdir("rux-ruby") do
      puts "Running rux-ruby specs with turbo_tests (excluding failing_examples_spec.rb)..."
      spec_files = Dir.glob("spec/**/*_spec.rb").reject { |f| f.include?("failing_examples_spec.rb") }

      sh "bundle exec turbo_tests #{spec_files.join(" ")}"
    end
  end

  desc "Run test_app specs using rux"
  task test_app: [:build] do
    Dir.chdir("test_app") do
      puts "Running test_app specs with rux..."

      # Use rux from PATH if available, otherwise use relative path
      rux_command = system("which rux > /dev/null 2>&1") ? "rux" : "../rux/rux"

      sh rux_command.to_s
    end
  end

  desc "Run integration tests in root spec directory"
  task integration: %i[build spec]
end

# ========================================
# Lint Tasks
# ========================================
namespace :lint do
  desc "Run all linting (Go and Ruby)"
  task all: %i[go ruby]

  desc "Lint Go code"
  task :go do
    Dir.chdir("rux") do
      puts "Running go fmt..."
      sh "go fmt ./..."

      puts "Running go vet..."
      sh "go vet ./..."

      # Run golint if available
      if system("which golint > /dev/null 2>&1")
        puts "Running golint..."
        sh "golint ./..."
      else
        puts "Note: golint not found, skipping (install with: go install golang.org/x/lint/golint@latest)"
      end
    end
  end

  desc "Lint Ruby code with Standard"
  task :ruby do
    if defined?(Rake::Task["standard"])
      Rake::Task["standard"].invoke
    else
      puts "Standard gem not found, skipping Ruby linting"
    end
  end

  desc "Fix Ruby linting issues automatically"
  task :ruby_fix do
    Rake::Task["standard:fix"].invoke
  end
  task fix: [:ruby_fix]
end

# ========================================
# Build Tasks
# ========================================
desc "Build the rux Go binary"
task build: :build_release
#   Dir.chdir("rux") do
#     puts "Building rux..."
#     sh "go build -o rux ."
#     puts "Binary created at rux/rux"
#   end
# end

desc "Build the rux Go binary with version information"
task :build_release do
  Dir.chdir("rux") do
    # Read version from VERSION file or use environment variable
    base_version = if ENV["VERSION"]
      ENV["VERSION"]
    elsif File.exist?("VERSION")
      File.read("VERSION").strip
    else
      raise "No version found - set via VERSION env var or VERSION file"
    end

    # Ensure version starts with 'v'
    version = base_version.start_with?("v") ? base_version : "v#{base_version}"

    # Get git commit (short hash)
    commit = `git rev-parse --short HEAD`.chomp

    # Get current timestamp in Go pseudo-version format (YYYYMMDD-HHMM)
    timestamp = Time.now.utc.strftime("%Y%m%d-%H%M")

    # Build pseudo-version: v0.6.0-TIMESTAMP-GITREF
    full_version = "#{version}-#{timestamp}-#{commit}"

    # Get build date
    date = Time.now.utc.strftime("%Y-%m-%d %H:%M:%S UTC")

    # Build with ldflags to embed version info
    ldflags = [
      "-X main.version=#{full_version}",
      "-X main.commit=#{commit}",
      "-X 'main.date=#{date}'",
      "-X main.builtBy=rake"
    ].join(" ")

    puts "Building rux with version: #{full_version}"
    sh %(go build -ldflags "#{ldflags}" -o rux .)
    puts "Binary created at rux/rux with version: #{full_version}"
  end
end

# ========================================
# Clean Tasks
# ========================================
desc "Clean Go build artifacts"
task :clean do
  Dir.chdir("rux") do
    rm_f "rux"
    puts "Cleaned Go build artifacts"
  end
end

# ========================================
# Benchmark Tasks
# ========================================
namespace :bench do
  desc "Run all benchmarks"
  task all: %i[rux_ruby test_app]

  desc "Run benchmarks on rux-ruby"
  task rux_ruby: [:build] do
    puts "Running benchmarks on rux-ruby..."
    sh "./script/bench ./rux-ruby"
  end

  desc "Run benchmarks on test_app"
  task test_app: [:build] do
    puts "Running benchmarks on test_app..."
    sh "./script/bench ./test_app"
  end
end

# ========================================
# CI Tasks
# ========================================
namespace :ci do
  desc "Run all CI checks"
  task all: ["lint:all", "test:all"]

  desc "Run CI checks for Go"
  task go: ["lint:go", "test:go"]

  desc "Run CI checks for Ruby"
  task ruby: ["lint:ruby", "test:ruby"]
end

# ========================================
# External Dependencies
# ========================================
namespace :deps do
  desc "Download watcher binary for file system monitoring"
  task :watcher do
    require "net/http"
    require "uri"
    require "fileutils"

    puts "Downloading watcher binary..."

    # Determine platform
    platform = case RUBY_PLATFORM
    when /darwin.*aarch64/, /darwin.*arm64/
      "aarch64-apple-darwin"
    when /darwin/
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
      
      URI.open(url) do |remote_file|
        File.open(temp_file, "wb") do |local_file|
          local_file.write(remote_file.read)
        end
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

# ========================================
# Convenience Tasks (top-level)
# ========================================
desc "Run all tests"
task test: ["test:all"]

desc "Run all linting"
task lint: ["lint:all"]
