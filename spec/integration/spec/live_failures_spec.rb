require "spec_helper"
require "timeout"

RSpec.describe "Live failure output" do
  def with_live_project(framework)
    Dir.mktmpdir("live-failures-", ROOT_PATH.join("tmp")) do |dir|
      project = Pathname.new(dir)
      if framework == "minitest"
        fixture = project_fixture!("minitest-outcomes")
        FileUtils.cp([fixture.join("Gemfile"), fixture.join("Gemfile.lock")], project)
        project.join("test").mkpath
        project.join("test/failure_test.rb").write(<<~TEST)
          require "minitest/autorun"
          class LiveFailureTest < Minitest::Test
            def test_fails
              flunk "LIVE_ASSERTION"
            end
            def test_errors
              raise "LIVE_ERROR"
            end
          end
        TEST
        project.join("test/blocked_test.rb").write(<<~TEST)
          require "minitest/autorun"
          class BlockedTest < Minitest::Test
            def test_waits
              deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + 15
              sleep 0.02 until File.exist?("release") || Process.clock_gettime(Process::CLOCK_MONOTONIC) > deadline
              assert File.exist?("release"), "live failures never arrived"
            end
          end
        TEST
      else
        project.join("spec").mkpath
        project.join("spec/failure_spec.rb").write(<<~TEST)
          RSpec.describe "Live failure" do
            2.times { |n| it("fails \#{n}") { expect(true).to be false } }
          end
        TEST
        project.join("spec/blocked_spec.rb").write(<<~TEST)
          RSpec.describe "Blocked" do
            it "waits" do
              deadline = Process.clock_gettime(Process::CLOCK_MONOTONIC) + 15
              sleep 0.02 until File.exist?("release") || Process.clock_gettime(Process::CLOCK_MONOTONIC) > deadline
              expect(File.exist?("release")).to be true
            end
          end
        TEST
      end
      yield project
    end
  end

  ["rspec", "minitest"].each do |framework|
    it "streams #{framework} failures through a pipe before another worker finishes" do
      with_live_project(framework) do |project|
        Bundler.with_unbundled_env do
          Open3.popen3(plur_binary, "--use", framework, "-n", "2", "--color=never", chdir: project) do |stdin, stdout, stderr, process|
            stdin.close
            live = +""
            begin
              Timeout.timeout(10) do
                loop do
                  live << stdout.readline
                  break if live.lines.grep(/\A(?:rspec |LiveFailureTest#)/).size == 2
                end
              end
              expect(process).to be_alive
              expect(live).not_to include("Finished in", "Failures:", "PLUR_JSON:")
            ensure
              project.join("release").write("continue")
            end
            rest = Timeout.timeout(10) { stdout.read }
            errors = stderr.read
            expect(process.value.exitstatus).to eq(1), live + rest + errors

            if framework == "rspec"
              live_lines = live.lines.grep(/\Arspec /).map(&:chomp)
              expect(live_lines.size).to eq(2)
              expect(live_lines).to all(match(/failure_spec\.rb\[1:[12]\]/))
              live_lines.each { |line| expect(rest).to include(line) }
              expect(rest).to include("3 examples, 2 failures")
            else
              expect(live).to include("LiveFailureTest#test_fails [test/failure_test.rb:4]")
              expect(live).to include("LiveFailureTest#test_errors\n")
              expect(rest).to include("LiveFailureTest#test_fails [test/failure_test.rb:4]")
              expect(rest).to include("3 runs, 2 assertions, 1 failure, 1 error, 0 skips")
              expect(rest).not_to include("Failed examples:")
            end
          end
        end
      end
    end
  end

  it "keeps live identities out of progress mode" do
    result = chdir(project_fixture!("minitest-failures")) do
      Bundler.with_unbundled_env { run_plur_allowing_errors("-f", "progress", "--color=never", "-n", "2") }
    end
    before_details = result.out.split("Failures:").first
    expect(before_details).not_to include("MixedResultsTest#", "ArrayOperationsTest#")
    expect(result.exit_status).to eq(1)
  end
end
