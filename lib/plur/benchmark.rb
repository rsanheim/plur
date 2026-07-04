require "json"
require "fileutils"
require "time"
require "open3"
require "bundler"
require "yaml"
require "socket"
require "tmpdir"
require "etc"
require_relative "../plur"

module Plur
  module Benchmark
    # Runs each pinned target in the manifest through one hyperfine invocation and
    # writes a pristine hyperfine.json + enriched result.json + output.log + one
    # append-only index.jsonl row per result element. Commands reach hyperfine
    # verbatim as an argv array (never a shell), so `{VAR}` is substituted by
    # hyperfine, not Ruby. See the recurring-benchmarks plan in plur-internal.
    class SuiteRunner
      attr_reader :run_id

      def initialize(manifest_path:, out_root:, trigger: "local", host: nil, only: nil, run_report: true, report_cmd: nil)
        @plur_root = Dir.pwd
        @manifest_path = File.expand_path(manifest_path)
        @out_root = File.expand_path(out_root)
        @trigger = trigger
        @only = only
        @run_report = run_report
        @report_cmd = report_cmd || File.join(@plur_root, "script", "bench-report")
        @host_slug = (host || ENV["BENCH_HOST"] || Socket.gethostname.split(".").first)
          .downcase.gsub(/[^a-z0-9_-]/, "-")
        @timestamp = ENV["BENCH_TIMESTAMP"] || Time.now.utc.strftime("%Y%m%d-%H%M%S")
        @defaults = {}
      end

      def run
        check_plur_installed!
        projects = load_manifest
        plur = capture_plur
        host = capture_host
        @run_id = "#{@timestamp}-#{plur["commit"].to_s[0, 8]}"

        host_dir = File.join(@out_root, host["name"])
        run_dir = File.join(host_dir, "runs", @run_id)
        index_path = File.join(host_dir, "index.jsonl")
        FileUtils.mkdir_p(run_dir)

        log "run_id=#{@run_id} host=#{host["name"]} trigger=#{@trigger}"
        log "plur=#{plur["version"]} out=#{@out_root}"

        started_at = Time.now.utc.iso8601
        status = "ok"
        write_run_json(run_dir, status: "running", started_at: started_at,
          finished_at: nil, plur: plur, host: host, projects: projects)
        begin
          projects.each do |project|
            benchmark_project(project, run_dir: run_dir, index_path: index_path,
              plur: plur, host: host)
            run_report_hook(host_dir) # incremental: refresh while a slow target runs
          end
        rescue => e
          status = "failed"
          warn "[bench-suite] FAILED: #{e.class}: #{e.message}"
          raise
        ensure
          write_run_json(run_dir, status: status, started_at: started_at,
            finished_at: Time.now.utc.iso8601, plur: plur, host: host, projects: projects)
          run_report_hook(host_dir)
        end

        log "done: #{run_dir}"
        run_dir
      end

      private

      def load_manifest
        data = YAML.safe_load_file(@manifest_path)
        @defaults = data["defaults"] || {}
        projects = data.fetch("projects")
        projects = projects.select { |p| p["name"] == @only } if @only
        raise "no projects matched #{@only.inspect} in #{@manifest_path}" if projects.empty?
        projects
      end

      def benchmark_project(project, run_dir:, index_path:, plur:, host:)
        name = project.fetch("name")
        log "=== #{name} (#{project["repo"]} @ #{project["ref"]}) ==="

        run_in = provision(project)
        target_sha = git(["rev-parse", "HEAD"], checkout_dir(project))

        # When a target lists plur-tags, also benchmark those published releases
        # (HEAD + each tag) as a `plur_ref` parameter-list, comparing versions.
        plur_refs = resolve_plur_tags(project)
        plur_refs = plur_refs.empty? ? nil : ["HEAD", *plur_refs]
        plur_bindir = plur_refs && provision_plur_binaries(plur_refs)

        proj_dir = File.join(run_dir, name)
        FileUtils.mkdir_p(proj_dir)
        hyperfine_json = File.join(proj_dir, "hyperfine.json")
        output_log = File.join(proj_dir, "output.log")

        argv = build_hyperfine_argv(project, hyperfine_json, plur_refs:, plur_bindir:)
        log "  cwd=#{run_in}"
        log "  #{argv.join(" ")}"

        ok = nil
        Bundler.with_unbundled_env do
          File.open(output_log, "w") do |f|
            ok = system(*argv, chdir: run_in, out: f, err: [:child, :out])
          end
        end

        unless ok
          tail = begin
            File.readlines(output_log).last(25).join
          rescue
            ""
          end
          raise "hyperfine failed for #{name} (non-zero exit). Tail of output.log:\n#{tail}"
        end
        unless File.exist?(hyperfine_json) && File.size(hyperfine_json) > 0
          raise "hyperfine produced no JSON at #{hyperfine_json}"
        end

        results = JSON.parse(File.read(hyperfine_json)).fetch("results")
        write_result_json(proj_dir, project, target_sha, plur, host, results)
        append_index_rows(index_path, project, target_sha, plur, host, results)
        log "  wrote #{proj_dir} (#{results.length} result element(s))"
      end

      def checkout_dir(project)
        File.join(@out_root, "targets", project.fetch("name"))
      end

      def provision(project)
        dir = checkout_dir(project)
        unless Dir.exist?(File.join(dir, ".git"))
          FileUtils.mkdir_p(File.dirname(dir))
          clone(project.fetch("repo"), dir)
        end
        git!(["fetch", "--tags", "--quiet", "origin"], dir)
        git!(["checkout", "--quiet", "--detach", project.fetch("ref")], dir)

        run_in = project["subdir"] ? File.join(dir, project["subdir"]) : dir
        bundle_install(run_in)
        run_in
      end

      def clone(repo, dest)
        if tool_path("git-cache") && system("git-cache", "clone", repo, dest)
          return
        end
        run!("git", "clone", repo, dest)
      end

      def bundle_install(dir)
        Bundler.with_unbundled_env do
          run!("bundle", "config", "set", "--local", "path", "vendor/bundle", chdir: dir)
          next if system("bundle", "check", chdir: dir)
          run!("bundle", "install", "--jobs", "4", "--retry", "3", chdir: dir)
        end
      end

      # parameter-lists and commands fall back to manifest defaults, so a target
      # using the standard worker sweep + command is just name/repo/ref. When
      # plur_refs is set, sweep the plur binary too: the leading `plur` token in
      # each command is swapped for the per-ref binary built in provision step.
      def build_hyperfine_argv(project, hyperfine_json, plur_refs: nil, plur_bindir: nil)
        warmup = (project["warmup"] || @defaults["warmup"]).to_s
        runs = (project["runs"] || @defaults["runs"]).to_s

        argv = ["hyperfine", "--shell", "none", "--style", "basic",
          "--warmup", warmup, "--runs", runs]
        Array(project["parameter-lists"] || @defaults["parameter-lists"]).each do |pl|
          argv += ["--parameter-list", pl.fetch("var"), pl.fetch("values").to_s]
        end
        if plur_refs
          argv += ["--parameter-list", "plur_ref", plur_refs.join(",")]
          argv += ["--command-name", "plur-{plur_ref}"]
        else
          Array(project["command-name"]).each { |n| argv += ["--command-name", n] }
        end
        argv += ["--setup", project["setup"]] if project["setup"]
        argv << "--ignore-failure" if project["ignore-failure"]
        argv += ["--export-json", hyperfine_json]
        commands = Array(project["commands"] || @defaults.fetch("commands"))
        commands = commands.map { |c| c.sub(/\Aplur(?=\s|$)/, "#{plur_bindir}/plur-{plur_ref}") } if plur_refs
        argv + commands
      end

      # plur-tags (per-project or defaults) names published releases to benchmark
      # alongside HEAD. "latest" resolves to the newest non-prerelease tag.
      def resolve_plur_tags(project)
        Array(project["plur-tags"] || @defaults["plur-tags"]).flat_map do |tag|
          (tag == "latest") ? Array(latest_plur_release) : [tag]
        end.compact.uniq
      end

      def latest_plur_release
        capture("git", "ls-remote", "--tags", "--refs", "https://github.com/rsanheim/plur.git")
          .lines.filter_map { |l| l[%r{refs/tags/(\S+)}, 1] }
          .reject { |t| t.include?("-") }
          .max_by { |t| Gem::Version.new(t.delete_prefix("v")) }
      end

      # Place a plur binary per ref at <out>/plur-bin/plur-<ref>: HEAD is the
      # installed binary; tags are the published release downloads (cached).
      def provision_plur_binaries(refs)
        bindir = File.join(@out_root, "plur-bin")
        FileUtils.mkdir_p(bindir)
        refs.each do |ref|
          dest = File.join(bindir, "plur-#{ref}")
          if ref == "HEAD"
            FileUtils.ln_sf(tool_path("plur"), dest)
          elsif !File.exist?(dest)
            download_plur_release(ref, dest)
          end
        end
        bindir
      end

      def download_plur_release(tag, dest)
        arch = {"x86_64" => "amd64", "amd64" => "amd64", "arm64" => "arm64", "aarch64" => "arm64"}
          .fetch(capture("uname", "-m"))
        base = "plur_#{tag.delete_prefix("v")}_#{capture("uname", "-s").downcase}_#{arch}"
        url = "https://github.com/rsanheim/plur/releases/download/#{tag}/#{base}.tar.gz"
        tmp_parent = File.join(@plur_root, "tmp")
        FileUtils.mkdir_p(tmp_parent)
        Dir.mktmpdir("bench-plur-release-", tmp_parent) do |tmp|
          archive = File.join(tmp, "plur.tar.gz")
          run!("curl", "-fsSL", url, "-o", archive)
          run!("tar", "-xzf", archive, "-C", tmp, "--strip-components", "1", "#{base}/plur")
          FileUtils.mv("#{tmp}/plur", dest)
          File.chmod(0o755, dest)
        end
      end

      def target_meta(project, sha)
        {"repo" => project.fetch("repo"), "ref" => project.fetch("ref"),
         "sha" => sha, "subdir" => project["subdir"]}.compact
      end

      def write_result_json(proj_dir, project, target_sha, plur, host, results)
        record = {
          "schema_version" => 1,
          "run_id" => @run_id,
          "project" => project.fetch("name"),
          "target" => target_meta(project, target_sha),
          "plur" => plur,
          "host" => host["name"],
          "results" => results
        }
        File.write(File.join(proj_dir, "result.json"), JSON.pretty_generate(record))
      end

      def append_index_rows(index_path, project, target_sha, plur, host, results)
        FileUtils.mkdir_p(File.dirname(index_path))
        File.open(index_path, "a") do |f|
          results.each do |r|
            row = {
              "schema_version" => 1,
              "run_id" => @run_id,
              "project" => project.fetch("name"),
              "target" => target_meta(project, target_sha),
              "plur" => plur,
              "host" => host["name"],
              "command" => r["command"],
              "parameters" => r["parameters"],
              "mean" => r["mean"], "stddev" => r["stddev"], "median" => r["median"],
              "min" => r["min"], "max" => r["max"],
              "user" => r["user"], "system" => r["system"],
              "runs" => (r["times"] || []).length,
              "exit_codes" => r["exit_codes"]
            }
            f.puts(JSON.generate(row))
          end
        end
      end

      def write_run_json(run_dir, status:, started_at:, finished_at:, plur:, host:, projects:)
        data = {
          "schema_version" => 1,
          "run_id" => @run_id,
          "trigger" => @trigger,
          "started_at" => started_at,
          "finished_at" => finished_at,
          "status" => status,
          "plur" => plur,
          "host" => host,
          "config" => @defaults,
          "projects" => projects.map { |p| p["name"] }
        }
        File.write(File.join(run_dir, "run.json"), JSON.pretty_generate(data))
      end

      # The report is a nicety; never let it fail the benchmark run.
      def run_report_hook(host_dir)
        return unless @run_report && File.exist?(@report_cmd)
        unless system(@report_cmd, "--host-dir", host_dir)
          warn "[bench-suite] report hook exited non-zero (non-fatal)"
        end
      rescue => e
        warn "[bench-suite] report hook error (non-fatal): #{e.message}"
      end

      def capture_plur
        raw = capture("plur", "--version")
        version = raw.include?("=") ? raw.split("=", 2).last.strip : raw.sub(/^plur\s+version\s*/i, "").strip
        commit = version[/([0-9a-f]{7,40})$/, 1] || git(["rev-parse", "HEAD"], @plur_root)
        {
          "version" => version,
          "commit" => commit,
          "branch" => git(["rev-parse", "--abbrev-ref", "HEAD"], @plur_root),
          "go_version" => tool_version("go", "version")&.slice(/go\d[\d.]*/) || "unknown"
        }
      end

      def capture_host
        os = capture("uname", "-s").downcase
        {
          "name" => @host_slug,
          "os" => os,
          "arch" => capture("uname", "-m"),
          "kernel" => capture("uname", "-r"),
          "cpu_model" => cpu_model(os),
          "cpu_cores_physical" => cpu_cores_physical(os),
          "nproc" => logical_cpus(os),
          "mem_total_bytes" => mem_total_bytes(os),
          "mem_available_bytes_before" => mem_available_bytes(os),
          "container_image" => ENV["BENCH_CONTAINER_IMAGE"],
          "ruby" => RUBY_VERSION,
          "hyperfine" => tool_version("hyperfine", "--version")&.split&.last,
          "load_avg_before" => load_avg(os)
        }
      end

      def tool_version(*argv)
        out = capture(*argv)
        out.empty? ? nil : out
      end

      def cpu_model(os)
        if os == "darwin"
          capture("sysctl", "-n", "machdep.cpu.brand_string")
        else
          line = File.foreach("/proc/cpuinfo").find { |l| l.start_with?("model name") }
          line&.split(":", 2)&.last&.strip
        end
      rescue
        nil
      end

      def logical_cpus(os)
        out = (os == "darwin") ? capture("sysctl", "-n", "hw.logicalcpu") : capture("nproc")
        out.to_i.positive? ? out.to_i : Etc.nprocessors
      rescue
        nil
      end

      def cpu_cores_physical(os)
        if os == "darwin"
          capture("sysctl", "-n", "hw.physicalcpu").to_i
        else
          cores = File.foreach("/proc/cpuinfo")
            .select { |l| l.start_with?("core id") }
            .map { |l| l.split(":").last.strip }.uniq.size
          cores.positive? ? cores : nil
        end
      rescue
        nil
      end

      def mem_total_bytes(os)
        if os == "darwin"
          capture("sysctl", "-n", "hw.memsize").to_i
        else
          line = File.foreach("/proc/meminfo").find { |l| l.start_with?("MemTotal") }
          line ? line.split[1].to_i * 1024 : nil
        end
      rescue
        nil
      end

      # macOS available memory is deferred (vm_stat math); nil is fine.
      def mem_available_bytes(os)
        return nil if os == "darwin"
        line = File.foreach("/proc/meminfo").find { |l| l.start_with?("MemAvailable") }
        line ? line.split[1].to_i * 1024 : nil
      rescue
        nil
      end

      def load_avg(os)
        if os == "darwin"
          capture("sysctl", "-n", "vm.loadavg").scan(/[\d.]+/).first(3).map(&:to_f)
        else
          File.read("/proc/loadavg").split.first(3).map(&:to_f)
        end
      rescue
        []
      end

      def git(args, dir)
        capture("git", "-C", dir, *args)
      end

      def git!(args, dir)
        run!("git", "-C", dir, *args)
      end

      def capture(*argv, chdir: nil)
        opts = chdir ? {chdir: chdir} : {}
        output, status = Open3.capture2e(*argv, **opts)
        status.success? ? output.strip : ""
      rescue Errno::ENOENT
        ""
      end

      def run!(*argv, chdir: nil)
        opts = chdir ? {chdir: chdir} : {}
        system(*argv, **opts) || raise("command failed: #{argv.join(" ")}#{" (in #{chdir})" if chdir}")
      end

      def tool_path(name)
        ENV.fetch("PATH", "").split(File::PATH_SEPARATOR)
          .map { |dir| File.join(dir, name) }
          .find { |path| File.executable?(path) && !File.directory?(path) }
      end

      def check_plur_installed!
        return if tool_path("plur")
        raise "plur not found in PATH. Run `bin/rake install` first (CI does this before bench-suite)."
      end

      def log(msg)
        puts "[bench-suite] #{msg}"
      end
    end
  end
end
