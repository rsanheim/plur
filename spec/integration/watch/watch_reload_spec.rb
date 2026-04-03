require "spec_helper"
require "open3"
require "timeout"

RSpec.describe "plur watch reload", :skip_if_ci do
  it "re-enters startup through the same path when reload is requested with relative -C", pending: "timing-sensitive, needs stabilization" do
    Dir.mktmpdir do |tmpdir|
      project_dir = File.join(tmpdir, "project")
      FileUtils.mkdir_p(File.join(project_dir, "spec"))

      File.write(File.join(project_dir, "spec", "example_spec.rb"), <<~RUBY)
        RSpec.describe "Example" do
          it "passes" do
            expect(true).to be true
          end
        end
      RUBY

      File.write(File.join(project_dir, "Gemfile"), "source 'https://rubygems.org'\ngem 'rspec'\n")
      File.write(File.join(project_dir, ".plur.toml"), <<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["bin/rspec"]
        target_pattern = "spec/**/*_spec.rb"
      TOML

      out = +""
      err = +""

      Dir.chdir(tmpdir) do
        Open3.popen3(plur_binary, "-C", "project", "--debug", "watch", "run", "--timeout", "4") do |stdin, stdout, stderr, wait_thr|
          Timeout.timeout(8) do
            reloaded = false

            until err.scan("plur watch starting!").count >= 2
              ready = IO.select([stdout, stderr], nil, nil, 0.1)
              next unless ready

              ready[0].each do |io|
                data = io.read_nonblock(4096)
                ((io == stdout) ? out : err) << data
              rescue IO::WaitReadable, EOFError
              end

              if !reloaded && err.include?("s/self/live@")
                stdin.puts("reload")
                stdin.close
                reloaded = true
              end
            end
          end

          Process.kill("TERM", wait_thr.pid)
          wait_thr.value
        end
      end

      expect(err.scan("plur watch starting!").count).to be >= 2
      expect(err).not_to include("failed to change directory")
    end
  end
end
