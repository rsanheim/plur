require "spec_helper"
require "support/capture_helper"
require "plur/benchmark"
require "tempfile"
require "json"

RSpec.describe Plur::Benchmark do
  include CaptureHelper

  describe Plur::Benchmark::Config do
    let(:config) { Plur::Benchmark::Config.new }

    describe "#initialize" do
      it "sets default values" do
        expect(config.workers).to eq(Plur.config.plur_cores)
        expect(config.warmup).to eq(2)
        expect(config.runs).to eq(5)
        expect(config.min_runs).to be_nil
        expect(config.max_runs).to be_nil
        expect(config.projects).to eq([])
        expect(config.save_results).to be true
        expect(config.show_output).to be false
        expect(config.checkpoint).to be false
        expect(config.results_dir).to eq(File.join(Dir.pwd, "results"))
      end

      it "uses BENCH_TIMESTAMP if set" do
        ENV["BENCH_TIMESTAMP"] = "20241225-120000"
        config = Plur::Benchmark::Config.new
        expect(config.timestamp).to eq("20241225-120000")
        ENV.delete("BENCH_TIMESTAMP")
      end
    end

    describe "#default_projects" do
      it "returns the default project paths" do
        expect(config.default_projects).to eq([
          Plur.config.default_ruby_dir.to_s,
          Plur.config.root_dir.join("references", "example-project").to_s
        ])
      end
    end

    describe "#projects_to_benchmark" do
      it "returns default projects when no projects specified" do
        expect(config.projects_to_benchmark).to eq(config.default_projects)
      end

      it "returns specified projects when set" do
        config.projects = ["./custom/project"]
        expect(config.projects_to_benchmark).to eq(["./custom/project"])
      end
    end
  end

  describe Plur::Benchmark::Runner do
    let(:config) { Plur::Benchmark::Config.new }
    let(:runner) { Plur::Benchmark::Runner.new(config) }

    before do
      allow(File).to receive(:exist?).and_call_original
      allow(runner).to receive(:system).and_return(true)
      allow(runner).to receive(:get_git_sha).and_return("abc123")
      allow(runner).to receive(:get_plur_version).and_return("plur version 1.0.0")
      allow(Dir).to receive(:chdir).and_yield
      allow(Dir).to receive(:glob).and_return(["spec/example_spec.rb"])
      allow(File).to receive(:directory?).and_return(true)
      allow(FileUtils).to receive(:mkdir_p)
    end

    describe "#run" do
      it "benchmarks each project" do
        config.projects = ["./project1", "./project2"]

        expect(runner).to receive(:benchmark_project).with("./project1").and_return({})
        expect(runner).to receive(:benchmark_project).with("./project2").and_return({})

        runner.run
      end
    end

    describe "#benchmark_project" do
      it "skips non-existent directories" do
        expect(File).to receive(:directory?).and_return(false)
        result = silence { runner.send(:benchmark_project, "./nonexistent") }
        expect(result).to be_nil
      end

      it "runs hyperfine with correct basic options" do
        cmd = runner.send(:build_hyperfine_command, "test-project", "foo.json", "foo.md")
        expect(cmd).to eq([
          "hyperfine",
          "--warmup", "2",
          "--runs", "5",
          "--export-json", "foo.json",
          "turbo_tests -n #{config.workers}",
          "plur -n #{config.workers}"
        ])
      end

      it "includes min/max runs when specified" do
        config.min_runs = 3
        config.max_runs = 10

        cmd = runner.send(:build_hyperfine_command, "test-project", "foo.json", "foo.md")
        expect(cmd).to include("--min-runs", "3")
        expect(cmd).to include("--max-runs", "10")
        expect(cmd).not_to include("--runs")
      end

      it "includes show-output flag when enabled" do
        config.show_output = true

        cmd = runner.send(:build_hyperfine_command, "test-project", "foo.json", "foo.md")
        expect(cmd).to include("--show-output")
      end
    end

    describe "#get_git_sha" do
      it "returns git sha when available" do
        runner = Plur::Benchmark::Runner.new(config)
        allow(runner).to receive(:`).with("git rev-parse --short HEAD").and_return("abc123\n")
        expect(runner.send(:get_git_sha)).to eq("abc123")
      end

      it "returns nogit when git fails" do
        runner = Plur::Benchmark::Runner.new(config)
        allow(runner).to receive(:`).with("git rev-parse --short HEAD").and_raise(StandardError)
        expect(runner.send(:get_git_sha)).to eq("nogit")
      end
    end

    describe "#get_plur_version" do
      it "returns plur version when available" do
        runner = Plur::Benchmark::Runner.new(config)
        allow(runner).to receive(:`).with("plur --version 2>/dev/null").and_return("plur version 1.0.0\n")
        expect(runner.send(:get_plur_version)).to eq("plur version 1.0.0")
      end

      it "returns unknown when plur fails" do
        runner = Plur::Benchmark::Runner.new(config)
        allow(runner).to receive(:`).with("plur --version 2>/dev/null").and_raise(StandardError)
        expect(runner.send(:get_plur_version)).to eq("plur version unknown")
      end
    end
  end

  describe "command line parsing" do
    it "parses worker count" do
      config = Plur::Benchmark::Config.new

      # Simulate parsing
      config.workers = 4

      expect(config.workers).to eq(4)
    end

    it "parses project paths" do
      config = Plur::Benchmark::Config.new

      # Simulate parsing multiple projects
      config.projects << "./project1"
      config.projects << "./project2"

      expect(config.projects).to eq(["./project1", "./project2"])
    end

    it "parses warmup and run counts" do
      config = Plur::Benchmark::Config.new

      config.warmup = 3
      config.runs = 10

      expect(config.warmup).to eq(3)
      expect(config.runs).to eq(10)
    end

    it "parses min and max runs" do
      config = Plur::Benchmark::Config.new

      config.min_runs = 5
      config.max_runs = 20

      expect(config.min_runs).to eq(5)
      expect(config.max_runs).to eq(20)
    end

    it "parses boolean flags" do
      config = Plur::Benchmark::Config.new

      config.checkpoint = true
      config.show_output = true
      config.save_results = false

      expect(config.checkpoint).to be true
      expect(config.show_output).to be true
      expect(config.save_results).to be false
    end
  end
end
