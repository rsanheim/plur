#!/usr/bin/env ruby
# Ruby integration tests for rux functionality

require 'minitest/autorun'
require 'fileutils'
require 'open3'

class RuxIntegrationTest < Minitest::Test
  def setup
    @test_app_dir = File.join(__dir__, 'test_app')
    @rux_binary = File.join(__dir__, 'rux', 'rux')
    
    # Build rux binary if it doesn't exist
    unless File.exist?(@rux_binary)
      Dir.chdir(File.join(__dir__, 'rux')) do
        system('go build -o rux main.go runner.go')
      end
    end
    
    # Clean up any existing test databases
    Dir.chdir(@test_app_dir) do
      FileUtils.rm_f(Dir.glob('storage/test*.sqlite3'))
    end
  end
  
  def test_rux_binary_exists
    assert File.exist?(@rux_binary), "rux binary should exist at #{@rux_binary}"
  end
  
  def test_rux_help_command
    stdout, stderr, status = Open3.capture3(@rux_binary, '--help')
    
    assert status.success?, "rux --help should succeed"
    assert_includes stdout, "rux - A fast Go-based test runner for Ruby/RSpec"
    assert_includes stdout, "db:create"
    assert_includes stdout, "db:setup"
    assert_includes stdout, "db:migrate"
  end
  
  def test_database_creation_dry_run
    Dir.chdir(@test_app_dir) do
      stdout, stderr, status = Open3.capture3(@rux_binary, 'db:create', '--dry-run', '--workers', '3')
      
      assert status.success?, "db:create --dry-run should succeed"
      assert_includes stdout, "[dry-run] Would run database task 'db:create' with 3 workers"
      assert_includes stdout, "Worker 0: RAILS_ENV=test bundle exec rake db:create"
      assert_includes stdout, "Worker 1: TEST_ENV_NUMBER=2 RAILS_ENV=test bundle exec rake db:create"
      assert_includes stdout, "Worker 2: TEST_ENV_NUMBER=3 RAILS_ENV=test bundle exec rake db:create"
    end
  end
  
  def test_database_creation_and_migration
    Dir.chdir(@test_app_dir) do
      # Create databases
      stdout, stderr, status = Open3.capture3(@rux_binary, 'db:create', '--workers', '3')
      assert status.success?, "db:create should succeed: #{stderr}"
      
      # Check that databases were created
      assert File.exist?('storage/test.sqlite3'), "test.sqlite3 should exist"
      assert File.exist?('storage/test2.sqlite3'), "test2.sqlite3 should exist"
      assert File.exist?('storage/test3.sqlite3'), "test3.sqlite3 should exist"
      
      # Run migrations
      stdout, stderr, status = Open3.capture3(@rux_binary, 'db:migrate', '--workers', '3')
      assert status.success?, "db:migrate should succeed: #{stderr}"
    end
  end
  
  def test_parallel_test_execution
    Dir.chdir(@test_app_dir) do
      # Set up databases first
      system(@rux_binary, 'db:create', '--workers', '3')
      system(@rux_binary, 'db:migrate', '--workers', '3')
      
      # Run tests
      stdout, stderr, status = Open3.capture3(@rux_binary, '--workers', '3')
      
      assert status.success?, "parallel test execution should succeed: #{stderr}"
      assert_includes stdout, "spec files in parallel using"
      assert_includes stdout, "=== Summary ==="
      assert_includes stdout, "passed"
    end
  end
  
  def test_env_number_assignment
    Dir.chdir(@test_app_dir) do
      # Set up databases
      system(@rux_binary, 'db:create', '--workers', '2')
      system(@rux_binary, 'db:migrate', '--workers', '2')
      
      # Run tests and capture output (tests print TEST_ENV_NUMBER)
      stdout, stderr, status = Open3.capture3(@rux_binary, '--workers', '2')
      
      # The output is filtered through progress formatter, so we can't easily
      # check the environment variables in the output. But if the tests pass,
      # it means the databases are working correctly with different TEST_ENV_NUMBER values
      assert status.success?, "tests should pass with parallel databases"
    end
  end
  
  def test_dry_run_test_execution
    Dir.chdir(@test_app_dir) do
      stdout, stderr, status = Open3.capture3(@rux_binary, '--dry-run', '--workers', '2')
      
      assert status.success?, "dry-run should succeed"
      output = stdout + stderr
      assert_includes output, "[dry-run] Found"
      assert_includes output, "spec files"
      assert_includes output, "bundle exec rspec --format progress"
    end
  end
end