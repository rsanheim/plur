#!/usr/bin/env ruby
# Setup script for Rails parallel testing with rux

puts "Setting up Rails app for parallel testing with rux..."

# Build rux binary
puts "Building rux binary..."
Dir.chdir('rux') do
  system('go build -o rux main.go runner.go') || abort("Failed to build rux")
end

# Setup Rails app
puts "Setting up Rails test app..."
Dir.chdir('test_app') do
  puts "Installing gems..."
  system('bundle install') || abort("Failed to install gems")
  
  puts "Setting up development database..."
  system('bundle exec rails db:create db:migrate') || abort("Failed to setup development database")
  
  puts "Setting up test databases with rux..."
  system('../rux/rux db:create --workers 3') || abort("Failed to create test databases")
  system('../rux/rux db:migrate --workers 3') || abort("Failed to migrate test databases")
  
  puts "Running tests to verify setup..."
  system('../rux/rux --workers 3') || abort("Tests failed")
end

puts "Running integration tests..."
system('ruby test_rux_integration.rb') || abort("Integration tests failed")

puts <<~SUCCESS

  ✅ Setup complete! 

  The Rails app is ready for parallel testing. You can now:

  1. Run tests in parallel:
     cd test_app && ../rux/rux --workers 3

  2. Setup databases:
     cd test_app && ../rux/rux db:create --workers 3
     cd test_app && ../rux/rux db:migrate --workers 3

  3. See what would run (dry-run):
     cd test_app && ../rux/rux --dry-run --workers 3

  4. Run integration tests:
     ruby test_rux_integration.rb

  Key features demonstrated:
  - TEST_ENV_NUMBER environment variable (worker 0 gets '', worker N gets 'N+1')
  - Parallel database creation and migration
  - Parallel test execution with isolated databases
  - Go-based performance improvements over Ruby-based parallel_tests

SUCCESS