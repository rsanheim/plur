require "rake"

begin
  require "bundler"
  require "standard/rake" if Gem::Specification.find_all_by_name("standard").any?
rescue LoadError
end

# Default task runs all checks
desc "Run all tests and linting"
task default: ["test:all", "lint:all"]

task install: [:build] do
  Dir.chdir("rux") do
    sh "go install ."
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
  desc "Run all tests (Go and Ruby)"
  task all: [:go, :ruby]

  desc "Run Go tests"
  task :go do
    Dir.chdir("rux") do
      puts "Running Go tests..."
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
  task ruby: [:build, :rux_ruby]

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
end

# ========================================
# Lint Tasks
# ========================================
namespace :lint do
  desc "Run all linting (Go and Ruby)"
  task all: [:go, :ruby]

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
    if defined?(Rake::Task["standard:fix"])
      Rake::Task["standard:fix"].invoke
    else
      puts "Standard gem not found, cannot auto-fix"
    end
  end
end

# ========================================
# Build Tasks
# ========================================
desc "Build the rux Go binary"
task :build do
  Dir.chdir("rux") do
    puts "Building rux..."
    sh "go build -o rux ."
    puts "Binary created at rux/rux"
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
  task all: [:rux_ruby, :test_app]

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
# Convenience Tasks (top-level)
# ========================================
desc "Run all tests"
task test: ["test:all"]

desc "Run all linting"
task lint: ["lint:all"]
