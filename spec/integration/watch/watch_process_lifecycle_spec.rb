require "spec_helper"

RSpec.describe "plur watch process lifecycle" do
  include PlurWatchHelper

  it "stops a manual run and its child on exit" do
    with_process_tree_project do |project|
      entered = false
      exited = false

      result = capture_plur_watch_process(dir: project, timeout: 10) do |process|
        if !entered && watch_ready?(process.err, process.ready_state, ready_dirs: :detected)
          process.stdin.puts("")
          entered = true
        elsif entered && !exited && process_pids(project).length == 2
          process.stdin.puts("exit")
          process.close_stdin
          exited = true
        end
      end

      expect(result).to be_success
      expect(result.out).to include("Exiting watch mode...")
      expect_processes_gone(project)
    end
  end

  it "stops a file-triggered run and its child on SIGTERM" do
    with_process_tree_project do |project|
      spec_file = project.join("spec/calculator_spec.rb")

      result = run_plur_watch(
        dir: project,
        timeout: 10,
        stop_on: ->(_process) { process_pids(project).length == 2 }
      ) do
        spec_file.write(spec_file.read + "\n# lifecycle spec\n")
      end

      expect(result).to be_success
      expect(result.out).to include("Received SIGTERM")
      expect_processes_gone(project)
    end
  end

  def with_process_tree_project
    with_temp_watch_project do |project|
      script = <<~RUBY.gsub("\n", "; ")
        child = nil
        trap('INT') { exit 130 }
        File.write('runner.pid', Process.pid.to_s)
        child = spawn('sleep', '60')
        File.write('child.pid', child.to_s)
        Process.wait(child)
      RUBY

      project.join(".plur.toml").write(<<~TOML)
        use = "rspec"

        [job.rspec]
        cmd = ["ruby", "-e", #{script.dump}]

        [[watch]]
        source = "spec/**/*_spec.rb"
        jobs = ["rspec"]
      TOML

      yield project
    ensure
      process_pids(project).each do |pid|
        Process.kill("KILL", pid) if process_alive?(pid)
      rescue Errno::ESRCH
        nil
      end
    end
  end

  def process_pids(project)
    %w[runner.pid child.pid].filter_map do |name|
      path = project.join(name)
      Integer(path.read) if path.exist?
    end
  end

  def expect_processes_gone(project)
    pids = process_pids(project)
    expect(pids.length).to eq(2)
    Timeout.timeout(3) { sleep 0.02 while pids.any? { |pid| process_alive?(pid) } }
  end

  def process_alive?(pid)
    Process.kill(0, pid)
    true
  rescue Errno::ESRCH
    false
  end
end
