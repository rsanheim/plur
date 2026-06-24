require "spec_helper"
require "plur/benchmark"

RSpec.describe Plur::Benchmark do
  describe Plur::Benchmark::SuiteRunner do
    # The real Phase 0 manifest is the contract; assert against it directly.
    let(:manifest) { Plur.config.root_dir.join("benchmarks", "projects.yml").to_s }
    let(:out_root) { Plur.config.root_dir.join("tmp", "bench-test").to_s }
    let(:runner) do
      Plur::Benchmark::SuiteRunner.new(manifest_path: manifest, out_root: out_root, host: "testhost")
    end

    describe "#build_hyperfine_argv" do
      it "assembles a no-shell argv with verbatim commands and parameter values" do
        project = runner.send(:load_manifest).first
        argv = runner.send(:build_hyperfine_argv, project, "/out/hyperfine.json")

        # {workers} must reach hyperfine untouched (the quoting footgun): the
        # command is one verbatim array element, never interpolated by Ruby.
        expect(argv).to eq([
          "hyperfine", "--shell", "none", "--style", "basic",
          "--warmup", "2", "--runs", "8",
          "--parameter-list", "workers", "4,8",
          "--export-json", "/out/hyperfine.json",
          "plur --workers {workers}"
        ])
      end

      it "adds optional pass-throughs only when present" do
        project = {
          "name" => "x", "repo" => "r", "ref" => "v1",
          "warmup" => 2, "runs" => 8,
          "command-name" => "plur-w{workers}",
          "setup" => "bundle install",
          "ignore-failure" => true,
          "commands" => ["plur"]
        }
        argv = runner.send(:build_hyperfine_argv, project, "/out/h.json")
        expect(argv).to include("--command-name", "plur-w{workers}")
        expect(argv).to include("--setup", "bundle install")
        expect(argv).to include("--ignore-failure")
      end

      it "keeps multiple benchmark commands as whole verbatim argv elements" do
        project = {
          "name" => "x", "repo" => "r", "ref" => "v1",
          "warmup" => 2, "runs" => 8,
          "parameter-lists" => [
            {"var" => "workers", "values" => "4,8"},
            {"var" => "mode", "values" => "cold,warm"}
          ],
          "commands" => [
            "plur --workers {workers} --mode {mode}",
            "bundle exec parallel_rspec -n {workers}"
          ]
        }

        argv = runner.send(:build_hyperfine_argv, project, "/out/h.json")

        expect(argv).to include("--shell", "none")
        expect(argv).to include("--parameter-list", "workers", "4,8")
        expect(argv).to include("--parameter-list", "mode", "cold,warm")
        expect(argv.last(2)).to eq([
          "plur --workers {workers} --mode {mode}",
          "bundle exec parallel_rspec -n {workers}"
        ])
        expect(argv).not_to include("{workers}")
        expect(argv).not_to include("{mode}")
      end

      it "sweeps the plur binary across refs when plur_refs is given" do
        project = {"name" => "x", "repo" => "r", "ref" => "v1", "warmup" => 2, "runs" => 8,
                   "commands" => ["plur --workers {workers}"]}
        argv = runner.send(:build_hyperfine_argv, project, "/o/h.json",
          plur_refs: ["HEAD", "v0.70.0"], plur_bindir: "/bin")
        expect(argv).to include("--parameter-list", "plur_ref", "HEAD,v0.70.0")
        expect(argv).to include("--command-name", "plur-{plur_ref}")
        # the leading `plur` token is swapped for the per-ref binary; args kept
        expect(argv.last).to eq("/bin/plur-{plur_ref} --workers {workers}")
      end
    end

    describe "#resolve_plur_tags" do
      it "returns literal tags (project over defaults), empty when unset" do
        runner.send(:load_manifest)
        expect(runner.send(:resolve_plur_tags, {"plur-tags" => ["v0.69.0"]})).to eq(["v0.69.0"])
        expect(runner.send(:resolve_plur_tags, {})).to eq([])
      end
    end

    describe "#load_manifest" do
      it "loads the pinned backspin target, inheriting the command from defaults" do
        projects = runner.send(:load_manifest)
        backspin = projects.find { |p| p["name"] == "backspin" }
        expect(backspin["ref"]).to eq("v0.12.0")
        # the standard command + worker sweep come from manifest defaults
        argv = runner.send(:build_hyperfine_argv, backspin, "/out/h.json")
        expect(argv).to include("--parameter-list", "workers", "4,8")
        expect(argv.last).to eq("plur --workers {workers}")
      end

      it "lets a target override the command (rack's minitest file pattern)" do
        rack = runner.send(:load_manifest).find { |p| p["name"] == "rack" }
        argv = runner.send(:build_hyperfine_argv, rack, "/out/h.json")
        expect(argv.last).to eq("plur test/spec_*.rb --use minitest --workers {workers}")
      end
    end

    describe "#capture_host" do
      it "captures live host facts as a JSON-ready hash" do
        host = runner.send(:capture_host)
        expect(host["name"]).to eq("testhost")
        expect(host["os"]).to match(/darwin|linux/)
        expect(host["nproc"]).to be > 0
        expect(host).to have_key("load_avg_before")
      end
    end
  end
end
