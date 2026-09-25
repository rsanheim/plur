require "spec_helper"

RSpec.describe "plur watch minitest run all" do
  include PlurWatchHelper

  def with_minitest_project
    Dir.mktmpdir("watch-minitest-", ROOT_PATH.join("tmp")) do |dir|
      project = Pathname.new(dir)
      fixture = project_fixture!("minitest-outcomes")
      FileUtils.cp([fixture.join("Gemfile"), fixture.join("Gemfile.lock")], project)
      FileUtils.cp_r(fixture.join(".bundle"), project) if fixture.join(".bundle").exist?
      project.join("test").mkpath
      %w[First Second Excluded].each do |name|
        project.join("test/#{name.downcase}_test.rb").write(<<~TEST)
          require "minitest/autorun"
          class #{name}Test < Minitest::Test
            def test_runs
              assert #{name != "Excluded"}
            end
          end
        TEST
      end
      project.join(".plur.toml").write(<<~TOML)
        use = "unit"

        [job.unit]
        framework = "minitest"
        cmd = ["bundle", "exec", "ruby", "-Itest"]
        target_pattern = "test/**/*_test.rb"
        exclude_patterns = ["test/excluded_test.rb"]

        [[watch]]
        source = "test/**/*_test.rb"
        jobs = ["unit"]
      TOML
      Bundler.with_unbundled_env { yield project }
    end
  end

  it "runs every matching file on Enter, honors excludes, and discovers new files on the next run" do
    with_minitest_project do |project|
      stage = 0
      result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
        if stage == 0 && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
          process.stdin.puts("")
          stage = 1
        elsif stage == 1 && process.err.include?("Finished job")
          project.join("test/new_test.rb").write(<<~TEST)
            require "minitest/autorun"
            class NewTest < Minitest::Test
              def test_new
                assert true
              end
            end
          TEST
          process.stdin.puts("")
          stage = 2
        elsif stage == 2 && process.out.include?("3 runs,")
          process.stdin.puts("exit")
          stage = 3
        end
      end

      expect(result).to be_success
      expect(result.out).to include("2 runs,", "3 runs,", "0 failures, 0 errors"), result.out + result.err
      expect(result.out).not_to include("ExcludedTest#", "PLUR_JSON:")
      expect(stage).to eq(3), result.out + result.err
    end
  end

  it "reports an empty suite instead of executing Ruby with no files" do
    with_minitest_project do |project|
      FileUtils.rm_f(Dir[project.join("test/*_test.rb")])
      result = run_plur_watch_interactive(commands: ["", "exit"], dir: project)

      expect(result.err).to include("no test files found")
      expect(result.out).not_to include("[plur] bundle exec ruby")
      expect(result).to be_success
    end
  end
end
