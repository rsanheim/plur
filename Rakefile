require "bundler/setup"
require "fileutils"
require "open3"
require "shellwords"
require_relative "lib/plur"

# Race detection: when PLUR_RACE is set, configure Go to build with race detector
if ENV["PLUR_RACE"] && ENV["PLUR_RACE"] != "0"
  # Add -race to GOFLAGS (idempotent)
  goflags = Shellwords.split(ENV["GOFLAGS"].to_s)
  unless goflags.include?("-race")
    goflags << "-race"
    ENV["GOFLAGS"] = goflags.join(" ")
  end

  # Race detector requires CGO
  ENV["CGO_ENABLED"] = "1"

  puts "[race] Race detection enabled: GOFLAGS=#{ENV["GOFLAGS"]} CGO_ENABLED=1"
end

begin
  require "standard/rake" if Gem::Specification.find_all_by_name("standard").any?
  require "rspec/core/rake_task" if Gem::Specification.find_all_by_name("rspec").any?
  RSpec::Core::RakeTask.new(:spec) if defined?(RSpec)
rescue LoadError
end

Dir.glob(Plur.config.root_dir.join("lib", "tasks", "*.rake")).each { |file| load file }

PLUR_CORES = Plur.config.plur_cores

# Default task runs all checks
desc "Run all tests, linting, and benchmark budgets"
task default: ["lint:all", "build", "test:all", "bench:cache"]

desc "Build the plur Go binary"
task build: ["vendor:download:current"] do
  puts "[build] Building plur"
  sh %(go build -mod=mod .)
  version = `./plur --version`.strip
  puts "[build] Binary created at ./plur with version: #{version}"
end

namespace :build do
  desc "Build binaries for all supported platforms"
  task all: ["vendor:download:all"] do
    puts "[build:all] Building plur for all platforms..."
    sh "goreleaser build --snapshot --clean"
  end
end

desc "Install plur globally to $GOBIN"
task :install do
  gobin = ENV.fetch("GOBIN") { File.join(`go env GOPATH`.strip, "bin") }
  FileUtils.mkdir_p(gobin)
  final = File.join(gobin, "plur")
  temp = File.join(gobin, "plur-new-#{Time.now.to_i}")

  sh %(goreleaser build --snapshot --single-target --clean -o #{temp} > /dev/null 2>&1)
  File.chmod(0o755, temp)
  File.rename(temp, final)
  puts "[install] Installed plur=#{final} with version: #{`#{final} --version`.strip}"

  path_plur, status = Open3.capture2("which", "plur")
  path_plur = path_plur.strip
  if status.success? && path_plur != final
    warn "[install] >>> WARNING: 'plur' on PATH resolves to #{path_plur}, not #{final}"
    warn "[install] >>> Your shell will use a different binary than what was just installed."
  end
end

namespace :test do
  desc "Run all tests (Go, Default Ruby, and full Ruby suite)"
  task all: ["test:go", "test:default_ruby", "test"]

  desc "Run Go tests"
  task go: ["build"] do
    puts "[test:go] Running go tests..."
    # Use -short in CI to skip slow complexity tests that are sensitive to system noise
    short_flag = ENV["CI"] ? "-short" : ""
    if ENV["VERBOSE"]
      sh "go test -mod=mod -v #{short_flag} ./...".squeeze(" ")
    else
      sh "go test -mod=mod #{short_flag} ./...".squeeze(" ")
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
    rails_dir = Plur.config.default_rails_dir.to_s
    Bundler.with_unbundled_env do
      Dir.chdir(rails_dir) { sh "bundle", "install", "--quiet" }
      sh "plur", "-C", rails_dir, "-n", PLUR_CORES.to_s, err: "/dev/null"
    end
  end

  desc "Run Ruby specs against all RSpec appraisals"
  task appraisals: :install do
    puts "[test:appraisals] Running specs against all RSpec appraisals..."
    sh "bundle exec appraisal rake test"
  end

  desc "Run Ruby specs against a specific appraisal (e.g., bin/rake test:appraisal[rspec-3.13.1])"
  task :appraisal, [:name] => :install do |_t, args|
    name = args[:name] || abort("Usage: bin/rake test:appraisal[rspec-3.13.1]")
    puts "[test:appraisal] Running specs with #{name}..."
    sh "bundle exec appraisal #{name} rake test"
  end
end

# Load budget for a large-suite runtime cache (>= 10K cached examples).
# The documented criterion is 50ms combined load+save
# (docs/plans/2026-05-22-runtime-cache-measurement-results.md), but save is
# fsync-dominated and tracks disk speed, not JSON speed, so the automated
# gate enforces the stable CPU-bound component: cache load. The runtime
# cache is loaded on every plur run, and goccy/go-json was adopted
# specifically because stdlib encoding/json decodes ~5x slower — stdlib
# blows this budget on every machine measured (41-74ms), goccy stays well
# under it (9-21ms). CI runners are slower shared VMs, so they get 1.5x
# laxity; that stays below stdlib's best observed time on any machine.
CACHE_LOAD_BUDGET_MS = Float(ENV.fetch("PLUR_CACHE_LOAD_BUDGET_MS") { ENV["CI"] ? "45" : "30" })

namespace :bench do
  desc "Run runtime-cache benchmarks and enforce the load budget"
  task :cache do
    if ENV["PLUR_RACE"] && ENV["PLUR_RACE"] != "0"
      puts "[bench:cache] Skipping benchmark budget under race detector (timing-sensitive)"
      next
    end

    puts "[bench:cache] Running runtime cache benchmarks..."
    output, status = Open3.capture2e(
      "go", "test", "-mod=mod", "-run", "^$",
      "-bench", "BenchmarkCache_(Load|Save)LargeRspecCache",
      "-benchmem", "-count", "3", "./internal/testruntime"
    )
    puts output
    abort "[bench:cache] >>> benchmark run failed" unless status.success?

    # Best-of-count per benchmark: the minimum is the least noise-sensitive
    # estimate of intrinsic speed on a shared or loaded machine.
    times = Hash.new { |h, k| h[k] = [] }
    output.scan(%r{^(BenchmarkCache_\w+)-\d+\s+\d+\s+([\d.]+) ns/op}) { |name, ns| times[name] << Float(ns) }
    load_ns = times["BenchmarkCache_LoadLargeRspecCache"].min
    save_ns = times["BenchmarkCache_SaveLargeRspecCache"].min
    abort "[bench:cache] >>> could not parse benchmark output" unless load_ns && save_ns

    load_ms = load_ns / 1_000_000.0
    save_ms = save_ns / 1_000_000.0
    puts format("[bench:cache] runtime cache load = %.1fms (budget %.0fms), save = %.1fms (informational)", load_ms, CACHE_LOAD_BUDGET_MS, save_ms)
    if load_ms > CACHE_LOAD_BUDGET_MS
      abort format(<<~MSG, load_ms, CACHE_LOAD_BUDGET_MS)
        [bench:cache] >>> Runtime cache load %.1fms exceeds the %.0fms large-suite budget.
        [bench:cache] >>> The cache is loaded on every plur invocation against a big project.
        [bench:cache] >>> If you changed the cache JSON implementation or schema, see the
        [bench:cache] >>> "go-json Parser Adoption" section of
        [bench:cache] >>> docs/plans/2026-05-22-runtime-cache-measurement-results.md before relaxing this budget.
      MSG
    end
  end
end

namespace :lint do
  desc "Run all linting (Go, Ruby, and Shell)"
  task all: %i[go ruby shell toolchain:check]

  desc "Lint Go code"
  task :go do
    puts "[lint:go] Running go fmt and go vet"
    sh "go", "fmt", "-mod=mod", "./..."
    sh "go vet -mod=mod ./..."
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

  desc "Lint shell scripts with shellcheck"
  task :shell do
    puts "[lint:shell] Running shellcheck"
    scripts = Dir["install.sh", "script/*"].select { |f| File.file?(f) && File.read(f, 64).start_with?("#!/") && File.read(f, 64).include?("sh") }
    sh "shellcheck", "--severity=warning", *scripts
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
