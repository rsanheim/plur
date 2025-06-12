require "bundler/setup"
require "fileutils"

begin
  require "standard/rake" if Gem::Specification.find_all_by_name("standard").any?
  require "rspec/core/rake_task" if Gem::Specification.find_all_by_name("rspec").any?
  RSpec::Core::RakeTask.new(:spec) if defined?(RSpec)
rescue LoadError
end

# Load all tasks from lib/tasks
Dir.glob(File.join(__dir__, "lib", "tasks", "*.rake")).each { |file| load file }

LOCAL_RUX = Pathname.new(__dir__).join("rux", "rux").expand_path.to_s.freeze

# Default task runs all checks
desc "Run all tests and linting"
task default: ["test:all", "lint:all"]

desc "Build the rux Go binary"
task :build do
  Dir.chdir("rux") do
    puts "Building rux"
    sh %(go build -mod=mod -o rux .)
    version = `./rux --version`.strip
    puts "Binary created at rux/rux with version: #{version}"
  end
end

desc "Build and install rux to GOPATH/bin"
task :install do
  Dir.chdir("rux") do
    sh %(go install -mod=mod .)
    gopath = `go env GOPATH`.chomp
    puts "Installed rux to #{gopath}/bin/ with version: #{`./rux --version`.strip}"
  end
end

# ========================================
# Test Tasks
# ========================================
# For Go tests:
#   - Default: minimal output (just pass/fail)
#   - VERBOSE=1 rake test:go: detailed test output
#   - Investigate gotestsum for better formatting
# ========================================
namespace :test do
  desc "Run all tests (Go, Ruby, and Integration)"
  task all: %i[go ruby integration]

  desc "Run Go tests"
  task :go do
    Dir.chdir("rux") do
      puts "running go tests..."

      sh "go mod download"

      if Dir.exist?("tmp")
        puts "Cleaning up tmp directory..."
        FileUtils.rm_rf("tmp")
        FileUtils.mkdir_p("tmp")
      end

      # Ensure we're in the right directory and can find embedded files
      raise "Cannot find rspec/formatter.rb - are we in the right directory?" unless File.exist?("rspec/formatter.rb")

      # Use standard output format (dots) unless verbose is requested
      # Use -mod=mod to ignore vendor directory and use module cache
      if ENV["VERBOSE"]
        sh "go test -mod=mod -v ./..."
      elsif system("which gotestsum > /dev/null 2>&1")
        # Use gotestsum for better formatting if available
        sh "gotestsum --format short -- -mod=mod ./..."
      else
        # This gives a cleaner output with just pass/fail
        sh "go test -mod=mod ./..."
      end
    end
  end

  desc "Run our default-ruby fixture project with rux"
  task ruby: %i[build default_ruby]

  desc "Run our default-ruby fixture project with rux"
  task :default_ruby do
    Dir.chdir("fixtures/projects/default-ruby") do
      puts "Running default-ruby specs with rux..."
      sh LOCAL_RUX
    end
  end

  desc "Run default-ruby specs using turbo_tests"
  task :default_ruby_turbo do
    Dir.chdir("fixtures/projects/default-ruby") do
      puts "Running default-ruby specs with turbo_tests..."

      sh "bundle exec turbo_tests"
    end
  end

  desc "Run default-rails specs using rux"
  task default_rails: [:build] do
    Dir.chdir("fixtures/projects/default-rails") do
      puts "Running default-rails specs with rux..."
      sh LOCAL_RUX
    end
  end

  desc "Run integration tests in root spec directory"
  task integration: :build do
    Dir.chdir("spec") do
      puts "Running ruby integration suite with rux..."
      sh LOCAL_RUX
    end
  end
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
    end
  end

  desc "Lint Ruby code with Standard"
  task :ruby do
    Rake::Task["standard"].invoke
  end

  desc "Fix Ruby linting issues automatically"
  task :ruby_fix do
    Rake::Task["standard:fix"].invoke
  end
  task fix: [:ruby_fix]
end

# ========================================
# Benchmark Tasks
# ========================================
namespace :bench do
  desc "Run all benchmarks"
  task all: %i[default_ruby rails8_app]

  desc "Run benchmarks on default-ruby"
  task default_ruby: [:build] do
    puts "Running benchmarks on default-ruby..."
    sh "./script/bench ./fixtures/projects/default-ruby"
  end

  desc "Run benchmarks on default-rails"
  task default_rails: [:build] do
    puts "Running benchmarks on default-rails..."
    sh "./script/bench ./fixtures/projects/default-rails"
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

# External Dependencies moved to lib/tasks/

# ========================================
# Convenience Tasks (top-level)
# ========================================
desc "Run all tests"
task test: ["test:all"]

desc "Run all linting"
task lint: ["lint:all"]
