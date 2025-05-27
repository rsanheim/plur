require 'rake'

desc "Run all checks (lint and test) for the Go rux implementation"
task :default => [:lint, :test]

desc "Lint the Go code"
task :lint do
  Dir.chdir('rux') do
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

desc "Test the Go code"
task :test do
  Dir.chdir('rux') do
    puts "Running go test..."
    sh "go test -v ./..."
  end
end

desc "Build the rux binary"
task :build do
  Dir.chdir('rux') do
    puts "Building rux..."
    sh "go build -o rux ."
    puts "Binary created at rux/rux"
  end
end

desc "Clean build artifacts"
task :clean do
  Dir.chdir('rux') do
    rm_f "rux"
    puts "Cleaned build artifacts"
  end
end

desc "Run benchmarks on test projects"
task :bench do
  puts "Running benchmarks on rux-ruby..."
  sh "./script/bench ./rux-ruby"
end

namespace :test do
  desc "Run rux-ruby specs using rux (excluding failing examples)"
  task :rux_ruby do
    Dir.chdir('rux-ruby') do
      puts "Running rux-ruby specs with rux (excluding failing_examples_spec.rb)..."
      # Find all spec files except failing_examples_spec.rb
      spec_files = Dir.glob("spec/**/*_spec.rb").reject { |f| f.include?("failing_examples_spec.rb") }
      
      # Use rux from PATH if available, otherwise use relative path
      rux_command = system("which rux > /dev/null 2>&1") ? "rux" : "../rux/rux"
      
      # Run rux with the specific files
      sh "#{rux_command} #{spec_files.join(' ')}"
    end
  end

  desc "Run rux-ruby specs using turbo_tests (excluding failing examples)"
  task :rux_ruby_turbo do
    Dir.chdir('rux-ruby') do
      puts "Running rux-ruby specs with turbo_tests (excluding failing_examples_spec.rb)..."
      spec_files = Dir.glob("spec/**/*_spec.rb").reject { |f| f.include?("failing_examples_spec.rb") }
      
      sh "bundle exec turbo_tests #{spec_files.join(' ')}"
    end
  end
end

desc "Build rux and run rux-ruby tests"
task :build_and_test => [:build, "test:rux_ruby"]