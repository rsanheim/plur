require "spec_helper"
require "tmpdir"

RSpec.describe "Plur stdout streaming" do
  describe "RSpec stdout behavior" do
    it "streams puts output in real-time" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "puts_spec.rb"), <<~RUBY)
          RSpec.describe 'Stdout test' do
            it 'prints to stdout' do
              puts "Hello from test"
              expect(true).to be true
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_plur("puts_spec.rb")

          expect(result.out).to include("Hello from test")
          expect(result.exit_status).to eq(0)
        end
      end
    end

    it "streams pp output in real-time" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "pp_spec.rb"), <<~RUBY)
          RSpec.describe 'PP test' do
            it 'pretty prints a hash' do
              pp({foo: "bar", baz: 123})
              expect(true).to be true
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_plur("pp_spec.rb")

          expect(result.out).to include("foo")
          expect(result.out).to include("bar")
          expect(result.exit_status).to eq(0)
        end
      end
    end

    it "shows stdout interleaved with progress indicators" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "interleaved_spec.rb"), <<~RUBY)
          RSpec.describe 'Interleaved output' do
            it 'prints before and after' do
              puts "BEFORE_ASSERTION"
              expect(true).to be true
              puts "AFTER_ASSERTION"
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_plur("interleaved_spec.rb")

          expect(result.out).to include("BEFORE_ASSERTION")
          expect(result.out).to include("AFTER_ASSERTION")
          # Should also have the dot progress indicator
          expect(result.out).to include(".")
          expect(result.exit_status).to eq(0)
        end
      end
    end

    it "handles empty puts gracefully" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "empty_puts_spec.rb"), <<~RUBY)
          RSpec.describe 'Empty puts' do
            it 'handles empty puts' do
              puts
              puts ""
              expect(true).to be true
            end
          end
        RUBY

        chdir(tmpdir) do
          result = run_plur("empty_puts_spec.rb")

          # Should complete successfully without crashing
          expect(result.out).to include("1 example, 0 failures")
          expect(result.exit_status).to eq(0)
        end
      end
    end
  end

  describe "stdout with multiple workers" do
    it "captures puts from all workers" do
      Dir.mktmpdir do |tmpdir|
        # Create multiple spec files that each print unique output
        3.times do |i|
          File.write(File.join(tmpdir, "worker_#{i}_spec.rb"), <<~RUBY)
            RSpec.describe 'Worker #{i} test' do
              it 'prints worker id' do
                puts "OUTPUT_FROM_WORKER_#{i}"
                sleep 0.1  # Small delay to ensure parallel execution
                expect(true).to be true
              end
            end
          RUBY
        end

        chdir(tmpdir) do
          result = run_plur("-n", "3", "worker_0_spec.rb", "worker_1_spec.rb", "worker_2_spec.rb")

          # All worker output should appear (order may vary due to parallelism)
          expect(result.out).to include("OUTPUT_FROM_WORKER_0")
          expect(result.out).to include("OUTPUT_FROM_WORKER_1")
          expect(result.out).to include("OUTPUT_FROM_WORKER_2")
          expect(result.exit_status).to eq(0)
        end
      end
    end
  end

  describe "Minitest stdout behavior" do
    it "does not stream stdout (avoiding duplication since consumed=false for all lines)" do
      minitest_fixture = project_fixture("minitest-success")

      chdir(minitest_fixture) do
        result = run_plur("test/calculator_test.rb")

        # Minitest tests should pass
        expect(result.out).to include("assertions")
        expect(result.out).to include("0 failures")
        expect(result.exit_status).to eq(0)

        # Key point: Minitest output appears through normal capture, not duplicated
        # We don't stream unconsumed stdout for Minitest because consumed=false for everything
      end
    end
  end
end
