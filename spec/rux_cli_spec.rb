require 'spec_helper'
require 'open3'

RSpec.describe 'Rux CLI behavior' do
  let(:rux_binary) { File.join(__dir__, '..', 'rux', 'rux') }
  let(:rux_ruby_dir) { File.join(__dir__, '..', 'rux-ruby') }

  before(:all) do
    # Build rux binary if needed
    rux_dir = File.join(__dir__, '..', 'rux')
    unless File.exist?(File.join(rux_dir, 'rux'))
      Dir.chdir(rux_dir) do
        system('go build -o rux', err: File::NULL)
      end
    end
  end

  describe 'dry-run functionality' do
    it 'runs dry-run with no arguments' do
      stdout, stderr, status = Open3.capture3(rux_binary, '--dry-run')
      
      expect(status.success?).to be true
      output = stdout + stderr
      expect(output).to include('[dry-run]')
    end

    it 'runs dry-run with specific spec file' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(rux_binary, '--dry-run', 'spec/calculator_spec.rb')
        
        expect(status.success?).to be true
        output = stdout + stderr
        expect(output).to include('[dry-run]')
        expect(output).to include('calculator_spec.rb')
      end
    end

    it 'runs dry-run with multiple spec files' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(
          rux_binary, 
          '--dry-run', 
          'spec/calculator_spec.rb', 
          'spec/rux_ruby_spec.rb'
        )
        
        expect(status.success?).to be true
        output = stdout + stderr
        expect(output).to include('[dry-run]')
        expect(output).to include('calculator_spec.rb')
        expect(output).to include('rux_ruby_spec.rb')
      end
    end

    it 'runs dry-run with --auto flag' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(
          rux_binary, 
          '--dry-run', 
          '--auto', 
          'spec/calculator_spec.rb'
        )
        
        expect(status.success?).to be true
        output = stdout + stderr
        expect(output).to include('[dry-run]')
      end
    end

    it 'runs dry-run with auto-discovery in rux-ruby' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(rux_binary, '--dry-run')
        
        expect(status.success?).to be true
        output = stdout + stderr
        expect(output).to include('[dry-run]')
        expect(output).to match(/Found \d+ spec files/)
      end
    end
  end

  describe 'actual test execution' do
    it 'runs all specs in parallel with auto-discovery' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(rux_binary)
        
        expect(status.success?).to be true
        expect(stdout).to include('Running')
        expect(stdout).to include('spec files in parallel')
        expect(stdout).to include('examples, 0 failures')
      end
    end

    it 'runs with --auto flag (bundle install + run)' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(rux_binary, '--auto')
        
        expect(status.success?).to be true
        output = stdout + stderr
        # Should either say "Installing dependencies" or skip if already installed
        expect(stdout).to include('examples, 0 failures')
      end
    end

    it 'runs specific spec file' do
      Dir.chdir(rux_ruby_dir) do
        stdout, stderr, status = Open3.capture3(rux_binary, 'spec/calculator_spec.rb')
        
        expect(status.success?).to be true
        expect(stdout).to include('examples, 0 failures')
      end
    end

    it 'provides interleaved output from parallel execution' do
      Dir.chdir(rux_ruby_dir) do
        stdout, _, status = Open3.capture3(rux_binary, '-n', '2')
        
        expect(status.success?).to be true
        # The dots should appear as tests complete
        expect(stdout).to match(/\.+/)
        expect(stdout).to include('examples, 0 failures')
      end
    end
  end
end