require "spec_helper"
require "tmpdir"
require "fileutils"

RSpec.describe "Plur --trace-output flag" do
  describe "with trace-output enabled" do
    it "prefixes stdout with file path" do
      Dir.mktmpdir do |tmpdir|
        spec_file = File.join(tmpdir, "traced_spec.rb")
        File.write(spec_file, <<~RUBY)
          RSpec.describe 'Traced test' do
            it 'prints to stdout' do
              puts "TRACE_ME_PLEASE"
              expect(true).to be true
            end
          end
        RUBY

        result = run_plur("-C", tmpdir, "--trace-output", "traced_spec.rb")

        expect(result.out).to include("[./traced_spec.rb]: TRACE_ME_PLEASE")
        expect(result.exit_status).to eq(0)
      end
    end

    it "adds newline before traced output for visual separation" do
      Dir.mktmpdir do |tmpdir|
        spec_file = File.join(tmpdir, "newline_spec.rb")
        File.write(spec_file, <<~RUBY)
          RSpec.describe 'Newline test' do
            it 'prints to stdout' do
              puts "SHOULD_HAVE_NEWLINE"
              expect(true).to be true
            end
          end
        RUBY

        result = run_plur("-C", tmpdir, "--trace-output", "newline_spec.rb")

        # Should have newline before the prefix (to separate from dots)
        expect(result.out).to match(/\n\[\.\/newline_spec\.rb\]:/)
        expect(result.exit_status).to eq(0)
      end
    end
  end

  describe "without trace-output (default)" do
    it "does not prefix output with file path" do
      Dir.mktmpdir do |tmpdir|
        spec_file = File.join(tmpdir, "untraced_spec.rb")
        File.write(spec_file, <<~RUBY)
          RSpec.describe 'Untraced test' do
            it 'prints to stdout' do
              puts "NO_TRACE"
              expect(true).to be true
            end
          end
        RUBY

        result = run_plur("-C", tmpdir, "untraced_spec.rb")

        expect(result.out).to include("NO_TRACE")
        expect(result.out).not_to include("[./untraced_spec.rb]:")
        expect(result.exit_status).to eq(0)
      end
    end
  end

  describe "with multiple workers" do
    it "traces output to correct files" do
      Dir.mktmpdir do |tmpdir|
        file_a = File.join(tmpdir, "file_a_spec.rb")
        File.write(file_a, <<~RUBY)
          RSpec.describe 'File A' do
            it 'prints from A' do
              puts "FROM_FILE_A"
              expect(true).to be true
            end
          end
        RUBY

        file_b = File.join(tmpdir, "file_b_spec.rb")
        File.write(file_b, <<~RUBY)
          RSpec.describe 'File B' do
            it 'prints from B' do
              puts "FROM_FILE_B"
              expect(true).to be true
            end
          end
        RUBY

        result = run_plur("-C", tmpdir, "--trace-output", "-n", "2", "file_a_spec.rb", "file_b_spec.rb")

        expect(result.out).to include("[./file_a_spec.rb]: FROM_FILE_A")
        expect(result.out).to include("[./file_b_spec.rb]: FROM_FILE_B")
        expect(result.exit_status).to eq(0)
      end
    end
  end
end
