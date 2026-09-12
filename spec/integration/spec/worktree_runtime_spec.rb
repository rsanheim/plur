require "spec_helper"

RSpec.describe "Runtime history across linked worktrees" do
  around_with_tmp_plur_home

  around do |example|
    Dir.mktmpdir("runtime-worktrees-", ROOT_PATH.join("tmp")) do |dir|
      @checkout = File.join(dir, "main")
      @worktree = File.join(dir, "linked")
      FileUtils.mkdir_p(@checkout)
      git("init", "--quiet", @checkout)
      [".", "apps/one", "apps/two"].each do |project|
        specs = File.join(@checkout, project, "spec")
        FileUtils.mkdir_p(specs)
        File.write(File.join(specs, "timed_spec.rb"), <<~SPEC)
          RSpec.describe "timed examples" do
            it("first") { expect(1).to eq(1) }
            it("second") { expect(2).to eq(2) }
          end
        SPEC
      end
      git("-C", @checkout, "add", ".")
      git("-C", @checkout, "-c", "user.name=Plur Test", "-c", "user.email=test@example.invalid", "commit", "--quiet", "-m", "Fixture")
      git("-C", @checkout, "worktree", "add", "--quiet", "--detach", @worktree)
      example.run
    end
  end

  def git(*args)
    out, err, status = Open3.capture3("git", *args)
    expect(status.success?).to be(true), "git #{args.join(" ")}: #{out}\n#{err}"
  end

  def cache_path(project, env: {})
    output = run_plur("-C", project, "doctor", env: env).out
    path = output[/^Runtime Data:\s+(.+)$/, 1]
    expect(path).not_to be_nil, output
    path
  end

  def run_specs(project, *args)
    run_plur("-C", project, "-n", "2", "spec/timed_spec.rb", *args)
  end

  %w[rails rake].each do |command|
    it "runs #{command} without looking up the runtime cache" do
      File.write(File.join(@checkout, ".plur.toml"), <<~TOML)
        [job.#{command}]
        cmd = ["sh", "-c", "echo task-ran"]
      TOML
      trace = File.join(File.dirname(@checkout), "git-trace.json")
      result = run_plur("-C", @checkout, command, "task", "-n", "1", env: {"GIT_TRACE2_EVENT" => trace})
      expect(result.out).to include("task-ran")
      expect(File.exist?(trace)).to be(false)
    end
  end

  it "shows loaded runtime counts in doctor and tolerates a corrupt cache" do
    run_specs(@checkout)
    path = cache_path(@checkout)
    expect(run_plur("-C", @checkout, "doctor").out).to include("1 files / 2 examples")
    File.write(path, "invalid JSON")
    expect(run_plur("-C", @checkout, "doctor").out).to include("(file exists)")
  end

  it "discovers Git identity in one command with optional locking disabled" do
    trace = File.join(File.dirname(@checkout), "git-trace.json")
    cache_path(@checkout, env: {"GIT_TRACE2_EVENT" => trace, "GIT_OPTIONAL_LOCKS" => "1"})
    commands = File.readlines(trace).map { |line| JSON.parse(line) }.select { |event| event["event"] == "start" }
    expect(commands.size).to eq(1)
    expect(commands.first.fetch("argv")).to include("--no-optional-locks", "rev-parse", "--git-common-dir", "--show-toplevel")
  end

  it "reuses recorded timings and the same filename in a linked worktree" do
    run_specs(@checkout)
    path = cache_path(@checkout)
    expect(File.exist?(path)).to be(true)
    expect(cache_path(@worktree)).to eq(path)
    result = run_specs(@worktree, "--dry-run", "--debug")
    expect(result.err).to include("Using runtime-based grouped execution")
    run_specs(@worktree)
    expect(Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))).to eq([path])
  end

  it "shares corresponding project subdirectories but keeps separate apps apart" do
    one = File.join(@checkout, "apps/one")
    two = File.join(@checkout, "apps/two")
    run_specs(one)
    expect(cache_path(File.join(@worktree, "apps/one"))).to eq(cache_path(one))
    expect(cache_path(two)).not_to eq(cache_path(one))
    expect(cache_path(@checkout)).not_to eq(cache_path(one))
  end

  it "keeps an independent clone's timing history separate" do
    clone = File.join(File.dirname(@checkout), "clone")
    git("clone", "--quiet", @checkout, clone)
    run_specs(@checkout)
    run_specs(clone)
    expect(cache_path(clone)).not_to eq(cache_path(@checkout))
    expect(Dir.glob(File.join(tmp_plur_home, "runtime", "*.json")).size).to eq(2)
  end

  it "does not use or change another worktree's example selectors" do
    run_specs(@checkout)
    path = cache_path(@checkout)
    original = JSON.parse(File.read(path))
    entry = original.fetch("files").fetch("spec/timed_spec.rb")
    # Matching timestamps must not bypass checkout isolation.
    source = File.stat(File.join(@worktree, "spec/timed_spec.rb"))
    entry["mtime_unix_nano"] = source.mtime.to_i * 1_000_000_000 + source.mtime.nsec
    File.write(path, JSON.generate(original))
    result = run_specs(@worktree, "--rspec-split", "--dry-run", "--debug")
    expect(result.err).not_to include("rspec-split applied")
    run_plur("-C", @worktree, "-n", "1", "spec/timed_spec.rb:2")
    expect(JSON.parse(File.read(path)).fetch("files")).to eq(original.fetch("files"))
  end

  it "preserves selectors when a focused run uses changed source" do
    run_specs(@checkout)
    path = cache_path(@checkout)
    original = JSON.parse(File.read(path)).fetch("files")
    File.open(File.join(@checkout, "spec/timed_spec.rb"), "a") { |file| file.puts("# changed source") }
    run_plur("-C", @checkout, "-n", "1", "spec/timed_spec.rb:2")
    expect(JSON.parse(File.read(path)).fetch("files")).to eq(original)
  end

  it "uses the same cache through a symlink to the checkout" do
    link = File.join(File.dirname(@checkout), "alias")
    File.symlink(@checkout, link)
    expect(cache_path(link)).to eq(cache_path(@checkout))
  end

  it "falls back to separate directory identities when Git cannot run" do
    env = {"PATH" => File.join(File.dirname(@checkout), "no-executables")}
    expect(cache_path(@checkout, env: env)).not_to eq(cache_path(@worktree, env: env))
    expect(cache_path(@checkout, env: env)).to eq(cache_path(@checkout, env: env))
  end

  it "keeps separate non-Git projects apart" do
    first = File.join(File.dirname(@checkout), "plain-one")
    second = File.join(File.dirname(@checkout), "plain-two")
    FileUtils.mkdir_p([first, second])
    expect(cache_path(first)).not_to eq(cache_path(second))
  end

  it "leaves a valid shared cache when linked worktrees finish concurrently" do
    commands = [@checkout, @worktree].map do |project|
      Thread.new { Open3.capture3(plur_binary, "-C", project, "-n", "1", "spec/timed_spec.rb") }
    end
    commands.each do |thread|
      out, err, status = thread.value
      expect(status.success?).to be(true), "#{out}\n#{err}"
    end
    paths = Dir.glob(File.join(tmp_plur_home, "runtime", "*.json"))
    expect(paths.size).to eq(1)
    expect(JSON.parse(File.read(paths.first)).fetch("files")).to include("spec/timed_spec.rb")
  end
end
