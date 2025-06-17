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

# Using 3 cores for medium/large resource class on CircleCI
RUX_CORES = ENV["CI"] ? 3 : 8

# Default task runs all checks
desc "Run all tests and linting"
task default: ["build", "test:all", "lint:ruby"]

desc "Build the rux Go binary"
task build: ["vendor:build", "lint:go"] do
  Dir.chdir("rux") do
    puts "[build] Building rux"
    sh %(go build -mod=mod -o rux .)
    version = `./rux --version`.strip
    puts "[build] Binary created at rux/rux with version: #{version}"
  end
end

desc "Build and install rux to GOPATH/bin"
task install: ["build"] do
  Dir.chdir("rux") do
    sh %(go install -mod=mod)
  end
  puts "[install] Installed rux with version: #{`rux --version`.strip}"
end

namespace :test do
  desc "Run all tests (Go, Ruby, and Integration)"
  task all: %i[go ruby integration]

  desc "Run Go tests"
  task go: ["build"] do
    Dir.chdir("rux") do
      puts "[test:go] Running go tests..."
      if ENV["VERBOSE"]
        sh "go test -mod=mod -v ./..."
      else
        sh "go test -mod=mod ./..."
      end
    end
  end

  desc "Run our default-ruby fixture project with rux"
  task ruby: %i[build default_ruby]

  desc "Run our default-ruby fixture project with rux"
  task :default_ruby do
    Dir.chdir("fixtures/projects/default-ruby") do
      puts "[test:default_ruby] Running default-ruby specs with rux..."
      sh "rux", "-n", RUX_CORES.to_s
    end
  end

  desc "Run default-ruby specs using turbo_tests"
  task :default_ruby_turbo do
    Dir.chdir("fixtures/projects/default-ruby") do
      puts "[test:default_ruby_turbo] Running default-ruby specs with turbo_tests..."
      sh "bundle exec turbo_tests"
    end
  end

  desc "Run default-rails specs using rux"
  task default_rails: [:build] do
    Dir.chdir("fixtures/projects/default-rails") do
      puts "[test:default_rails] Running default-rails specs with rux..."
      sh "rux", "-n", RUX_CORES.to_s
    end
  end

  desc "Run integration tests in root spec directory"
  task :integration do
    puts "[test:integration] Running ruby integration suite with rux..."
    sh "rux", "-n", RUX_CORES.to_s
  end
end

namespace :lint do
  desc "Run all linting (Go and Ruby)"
  task all: %i[go ruby]

  desc "Lint Go code"
  task :go do
    Dir.chdir("rux") do
      puts "[lint:go] Running go fmt and go vet"
      sh "go fmt -mod=mod ./..."
      sh "go vet -mod=mod ./..."
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

namespace :bench do
  desc "Run all benchmarks"
  task all: %i[default_ruby rails8_app]

  desc "Run benchmarks on default-ruby"
  task default_ruby: [:build] do
    puts "[bench:default_ruby] Running benchmarks on default-ruby..."
    sh "./script/bench ./fixtures/projects/default-ruby"
  end

  desc "Run benchmarks on default-rails"
  task default_rails: [:build] do
    puts "[bench:default_rails] Running benchmarks on default-rails..."
    sh "./script/bench ./fixtures/projects/default-rails"
  end
end

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
