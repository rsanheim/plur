require 'spec_helper'
require 'tmpdir'

RSpec.describe "Rux error handling" do
  let(:rux_binary) { File.join(__dir__, '..', 'rux', 'rux') }

  before do
    expect(File.exist?(rux_binary)).to be(true)
  end

  context "when JSON output is empty or malformed" do
    it "provides helpful error message instead of raw JSON parsing error" do
      Dir.mktmpdir do |tmpdir|
        # Create a minimal Rakefile that exits immediately
        rakefile_path = File.join(tmpdir, "Rakefile")
        File.write(rakefile_path, <<~RUBY)
          # This simulates a case where RSpec exits before generating JSON
          if ENV['RSPEC_JSON_OUTPUT']
            # Write empty file to simulate incomplete JSON
            File.write(ENV['RSPEC_JSON_OUTPUT'], '')
          end
          exit!(127)
        RUBY

        # Create a wrapper script that mimics rspec but exits early
        wrapper_path = File.join(tmpdir, "rspec")
        File.write(wrapper_path, <<~RUBY)
          #!/usr/bin/env ruby
          # Simulate RSpec failing before it can generate JSON
          exit(127)
        RUBY
        File.chmod(0755, wrapper_path)

        # Create a simple spec file
        spec_path = File.join(tmpdir, "test_spec.rb")
        File.write(spec_path, <<~RUBY)
          RSpec.describe "test" do
            it "never runs" do
              expect(1).to eq(1)
            end
          end
        RUBY

        # Create a Gemfile that will make bundle exec use our wrapper
        gemfile_path = File.join(tmpdir, "Gemfile")
        File.write(gemfile_path, <<~GEMFILE)
          source 'https://rubygems.org'
          # No rspec gem, so bundle exec rspec will fail
        GEMFILE

        # Run rux and capture output
        Dir.chdir(tmpdir) do
          output = `PATH=#{tmpdir}:$PATH #{rux_binary} test_spec.rb 2>&1`
          
          # Should show that something went wrong
          expect($?.exitstatus).not_to eq(0)
          
          # Should show helpful error message, not raw JSON error
          expect(output).to include("likely failed to start").or include("may have failed").or include("exit code")
          
          # Most importantly, should NOT show raw JSON parsing error
          expect(output).not_to include("failed to parse JSON: unexpected end of JSON input")
        end
      end
    end
  end

  context "when RSpec runs but JSON is malformed" do
    it "provides helpful error message for incomplete JSON" do
      Dir.mktmpdir do |tmpdir|
        # Create a working Gemfile
        gemfile_path = File.join(tmpdir, "Gemfile")
        File.write(gemfile_path, <<~GEMFILE)
          source 'https://rubygems.org'
          gem 'rspec', '~> 3.0'
        GEMFILE

        # Create a spec file that will cause RSpec to crash mid-execution
        spec_path = File.join(tmpdir, "crash_spec.rb")
        File.write(spec_path, <<~RUBY)
          RSpec.describe "crash test" do
            it "crashes during execution" do
              # This will cause RSpec to crash and potentially leave incomplete JSON
              exit!(1)  # Force exit without cleanup
            end
          end
        RUBY

        # Install gems first
        Dir.chdir(tmpdir) do
          system("bundle install", out: File::NULL, err: File::NULL)
          
          output = `#{rux_binary} crash_spec.rb 2>&1`
          
          # For this case, we expect either success (if RSpec handles the exit gracefully)
          # or a warning about JSON parsing (not a fatal error)
          expect(output).to include("Running 1 spec files")
          
          # Should not crash rux itself
          expect(output).to include("Finished in")
        end
      end
    end
  end

  context "when spec directory doesn't exist" do
    it "handles missing spec directory gracefully" do
      Dir.mktmpdir do |tmpdir|
        Dir.chdir(tmpdir) do
          output = `#{rux_binary} 2>&1`
          
          # Should handle gracefully and show appropriate message
          expect(output).to include("no spec files found").or include("ERROR")
        end
      end
    end
  end
end