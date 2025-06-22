require "json"
require "fileutils"
require "time"
require "open3"
require "pathname"
require_relative "../../plur"

module Plur
  module Benchmark
    class Config
      attr_accessor :workers, :warmup, :runs, :min_runs, :max_runs, :projects, :trace, :save_results,
        :show_output, :checkpoint, :results_dir, :timestamp

      def initialize
        @workers = Plur.config.rux_cores
        @warmup = 2
        @runs = 5
        @min_runs = nil
        @max_runs = nil
        @projects = []
        @trace = false
        @save_results = true
        @show_output = false
        @checkpoint = false
        @results_dir = File.join(Dir.pwd, "results")
        @timestamp = ENV["BENCH_TIMESTAMP"] || Time.now.utc.strftime("%Y%m%d-%H%M%S")
      end

      def default_projects
        [
          Plur.config.default_ruby_dir.to_s,
          Plur.config.root_dir.join("references", "example-project").to_s
        ]
      end

      def projects_to_benchmark
        @projects.empty? ? default_projects : @projects
      end
    end

    class Runner
      attr_reader :config, :git_sha, :rux_version, :original_dir

      def initialize(config)
        @config = config
        @original_dir = Dir.pwd
        check_local_rux_binary!
        @git_sha = get_git_sha
        @rux_version = get_rux_version
      end

      def run
        FileUtils.mkdir_p(config.results_dir)

        results = benchmark_projects

        if config.checkpoint
          create_checkpoint(results)
        end

        results
      end

      private

      def benchmark_projects
        config.projects_to_benchmark.map do |project_path|
          benchmark_project(project_path)
        end.compact
      end

      def benchmark_project(project_path)
        unless File.directory?(project_path)
          puts "Warning: Directory not found: #{project_path}"
          return nil
        end

        project_name = File.basename(project_path)
        puts "\n=== Benchmarking #{project_name} ==="

        Dir.chdir(project_path) do
          spec_count = Dir.glob("spec/**/*_spec.rb").count
          puts "Found #{spec_count} spec files"

          result = run_hyperfine(project_name)

          if config.trace
            analyze_traces(project_name)
          end

          result
        end
      end

      def run_hyperfine(project_name)
        results_path = Pathname.new(config.results_dir)
        json_file = results_path.join("#{config.timestamp}-#{git_sha}-#{project_name}.json").to_s
        markdown_file = results_path.join("#{config.timestamp}-#{git_sha}-#{project_name}.md").to_s

        rux_cmd = "#{Plur.config.local_rux_binary} -n #{config.workers}"
        rux_cmd += " --trace" if config.trace

        hyperfine_cmd = [
          "hyperfine",
          "--warmup", config.warmup.to_s
        ]

        if config.runs && !config.min_runs && !config.max_runs
          hyperfine_cmd += ["--runs", config.runs.to_s]
        else
          hyperfine_cmd += ["--min-runs", config.min_runs.to_s] if config.min_runs
          hyperfine_cmd += ["--max-runs", config.max_runs.to_s] if config.max_runs
        end

        hyperfine_cmd += ["--show-output"] if config.show_output

        if config.save_results
          hyperfine_cmd += ["--export-json", json_file]
          hyperfine_cmd += ["--export-markdown", markdown_file] if config.checkpoint
        end

        hyperfine_cmd += [
          "turbo_tests -n #{config.workers}",
          rux_cmd
        ]

        puts "Running benchmarks with #{config.workers} workers, #{config.warmup} warmup runs, #{config.runs} runs"
        puts "Rux version: #{rux_version}"
        puts "Tracing: ENABLED" if config.trace
        puts "===================="

        system(*hyperfine_cmd)

        if File.exist?(json_file)
          add_version_to_json(json_file)
          puts "\nResults saved to:\n  - #{json_file}"
          puts "  - #{markdown_file}" if File.exist?(markdown_file)

          # Return the result data
          {
            project: project_name,
            json_file: json_file,
            markdown_file: markdown_file,
            data: JSON.parse(File.read(json_file))
          }
        end
      end

      def add_version_to_json(json_file)
        data = JSON.parse(File.read(json_file))
        data["rux_version"] = rux_version
        File.write(json_file, JSON.pretty_generate(data))
      rescue => e
        puts "Warning: Could not add version to JSON: #{e.message}"
      end

      def analyze_traces(project_name)
        # Implementation for trace analysis
        # This would be ported from the bash script
      end

      def create_checkpoint(results)
        checkpoint = Checkpoint.new(config, results, git_sha, rux_version)
        checkpoint.create
      end

      def get_git_sha
        `git rev-parse --short HEAD`.strip
      rescue
        "nogit"
      end

      def get_rux_version
        `#{Plur.config.local_rux_binary} --version 2>/dev/null`.strip
      rescue
        "rux version unknown"
      end

      def check_local_rux_binary!
        unless File.exist?(Plur.config.local_rux_binary)
          puts <<~ERROR
            Error: Local rux binary not found at #{Plur.config.local_rux_binary}
            
            Please build rux first by running:
              bin/rake build
          ERROR
          exit 1
        end
      end
    end

    class Checkpoint
      attr_reader :config, :results, :git_sha, :rux_version

      def initialize(config, results, git_sha, rux_version)
        @config = config
        @results = results
        @git_sha = git_sha
        @rux_version = rux_version
      end

      def create
        write_json_summary if config.save_results
        write_markdown_summary
      end

      private

      def write_json_summary
        git_sha_long = begin
          `git rev-parse HEAD`.strip
        rescue
          git_sha
        end

        summary = {
          "commit" => git_sha_long,
          "timestamp" => config.timestamp,
          "rux_version" => rux_version,
          "projects" => results.map { |r| r[:project] if r }.compact
        }

        # Add metadata for each project
        results.each do |result|
          next unless result
          project_key = "#{result[:project]}_results"
          summary[project_key] = {
            "json_file" => result[:json_file],
            "markdown_file" => result[:markdown_file]
          }
        end

        results_path = Pathname.new(config.results_dir)
        json_file = results_path.join("#{config.timestamp}-#{git_sha}-summary.json")
        json_file.write(JSON.pretty_generate(summary))
        puts "\nCheckpoint summary saved to: #{json_file}"
      end

      def write_markdown_summary
        results_path = Pathname.new(config.results_dir)
        md_file = results_path.join("#{config.timestamp}-#{git_sha}-summary.md")

        md_file.open("w") do |f|
          f.puts "# Benchmark Summary"
          f.puts ""
          f.puts "- **Date**: #{config.timestamp} UTC"
          f.puts "- **Commit**: [#{git_sha}](https://github.com/rsanheim/rux-meta/commit/#{begin
            `git rev-parse HEAD`.strip
          rescue
            git_sha
          end})"
          f.puts "- **Rux Version**: #{rux_version}"
          f.puts ""
          f.puts "## Results"
          f.puts ""

          # Include the hyperfine-generated markdown for each project
          results.each do |result|
            next unless result && File.exist?(result[:markdown_file])

            f.puts "### #{result[:project]}"
            f.puts ""

            # Read hyperfine's markdown and include it
            hyperfine_md = File.read(result[:markdown_file])
            # Skip the header line if it exists
            lines = hyperfine_md.lines
            lines.shift if lines.first&.start_with?("|")
            f.puts lines.join

            # Add percentage comparison
            if result[:data] && result[:data]["results"]
              comparison = calculate_comparison(result[:data])
              f.puts comparison if comparison
            end
            f.puts ""
          end
        end

        puts "Summary Markdown saved to: #{md_file}"
        puts "\n#{md_file.read}"
      end

      def calculate_comparison(data)
        results = data["results"]
        return nil unless results && results.size >= 2

        turbo = results.find { |r| r["command"].include?("turbo_tests") }
        rux = results.find { |r| r["command"].include?("rux") }

        return nil unless turbo && rux

        diff_percent = ((rux["mean"] - turbo["mean"]) / turbo["mean"] * 100).round(1)
        status = (diff_percent > 0) ? "slower" : "faster"

        "\n**rux is #{diff_percent.abs}% #{status} than turbo_tests**"
      end
    end
  end
end
