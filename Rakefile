require "bundler/setup"
require "fileutils"
require_relative "lib/plur"

begin
  require "standard/rake" if Gem::Specification.find_all_by_name("standard").any?
  require "rspec/core/rake_task" if Gem::Specification.find_all_by_name("rspec").any?
  RSpec::Core::RakeTask.new(:spec) if defined?(RSpec)
rescue LoadError
end

Dir.glob(Plur.config.root_dir.join("lib", "tasks", "*.rake")).each { |file| load file }

PLUR_CORES = Plur.config.plur_cores

# Default task runs all checks
desc "Run all tests and linting"
task default: ["lint:all", "build", "test:all"]

desc "Build the plur Go binary"
task build: ["vendor:download:current"] do
  Dir.chdir(Plur.config.plur_dir) do
    puts "[build] Building plur"
    sh %(go build .)
    version = `./plur --version`.strip
    puts "[build] Binary created at plur/plur with version: #{version}"
  end
end

desc "Install plur globally to $GOBIN"
task :install do
  if ENV["CI"] && system("which plur")
    puts "[install] Plur already installed"
  else
    Dir.chdir(Plur.config.plur_dir) do
      gobin = ENV.fetch("GOBIN")
      final = File.join(gobin, "plur")
      temp = File.join(gobin, "plur-new-#{Time.now.to_i}")

      # We build to a temp file and then rename it to the final name to handle this atomically
      sh %(goreleaser build --snapshot --single-target --clean -o #{temp})
      File.chmod(0o755, temp)
      File.rename(temp, final)
    end
  end
  puts "[install] Installed plur with version: #{`plur --version`.strip}"
end

namespace :test do
  desc "Run all tests (Go, Default Ruby, and full Ruby suite)"
  task all: ["test:go", "test:default_ruby", "test"]

  desc "Run Go tests"
  task go: ["build"] do
    Dir.chdir(Plur.config.plur_dir) do
      puts "[test:go] Running go tests..."
      # Use -short in CI to skip slow complexity tests that are sensitive to system noise
      short_flag = ENV["CI"] ? "-short" : ""
      if ENV["VERBOSE"]
        sh "go test -mod=mod -v #{short_flag} ./...".squeeze(" ")
      else
        sh "go test -mod=mod #{short_flag} ./...".squeeze(" ")
      end
    end
  end

  desc "Run plur against default-ruby fixture project"
  task default_ruby: :install do
    puts "[test:default_ruby] Running default-ruby specs with plur..."
    sh "plur", "-C", Plur.config.default_ruby_dir.to_s, "-n", PLUR_CORES.to_s, err: "/dev/null"
  end

  desc "Run default-rails specs using plur"
  task default_rails: :install do
    puts "[test:default_rails] Running default-rails specs with plur..."
    sh "plur", "-C", Plur.config.default_rails_dir.to_s, "-n", PLUR_CORES.to_s, err: "/dev/null"
  end
end

namespace :lint do
  desc "Run all linting (Go and Ruby)"
  task all: %i[go ruby]

  desc "Lint Go code"
  task :go do
    Dir.chdir(Plur.config.plur_dir) do
      puts "[lint:go] Running go fmt and go vet"
      sh "go", "fmt", "-mod=mod", "./..."
      sh "go vet -mod=mod ./..."
    end
  end

  desc "Lint Ruby code with Standard"
  task :ruby do
    puts "[lint:ruby] Running StandardRB linter"
    Rake::Task["standard"].invoke
  end

  desc "Fix Ruby linting issues automatically"
  task :ruby_fix do
    Rake::Task["standard:fix"].invoke
  end
end

# ========================================
# Convenience Tasks (top-level)
# ========================================
desc "Run all Ruby specs"
task test: :install do
  puts "[test] Running all ruby specs with plur..."
  sh "plur", "-n", PLUR_CORES.to_s
end

desc "Run all linting"
task lint: ["lint:all"]
