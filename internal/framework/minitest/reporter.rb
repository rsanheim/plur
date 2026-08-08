# frozen_string_literal: true

# Plur's minitest integration, loaded via RUBYOPT before the test process
# starts. Replaces minitest's reporters with one that emits structured
# PLUR_JSON rows for plur to parse. With ProgressReporter and SummaryReporter
# out of the composite, nothing else writes progress characters to stdout, so
# any bare line on the pipe is test-written output by elimination.
#
# Inert when minitest isn't available (test-unit projects, plain ruby
# subprocesses spawned by tests).
begin
  require "minitest"
rescue LoadError
  return
end

require "json"

module Plur
  # Streams one JSON row per test to minitest's own output IO (options[:io],
  # captured before any test code can swap $stdout). StatisticsReporter
  # supplies the counting and the passed? exit-code contract; both have been
  # stable extension points since minitest 5.0.
  class MinitestReporter < Minitest::StatisticsReporter
    SEPARATOR = ENV["PLUR_FORMATTER_SEPARATOR"] || "PLUR_JSON:"
    # Placeholder failure number - plur renumbers globally across workers
    FAILURE_PLACEHOLDER = "‽"

    def start
      super
      output_row(type: :suite_started)
    end

    def record(result)
      super

      file, line = result.source_location if result.respond_to?(:source_location)
      file = file.delete_prefix("#{Dir.pwd}/") if file

      output_row(
        type: :test_result,
        code: result.result_code,
        id: "#{class_name_of(result)}##{result.name}",
        file_path: file,
        line_number: line,
        run_time: result.time,
        assertions: result.assertions
      )
    end

    def report
      super

      # Failure details in minitest's native format ("  1) Failure:\n..."),
      # with placeholder numbers since each worker only knows its own count.
      formatted = results.reject(&:skipped?).map { |r| "\n  #{FAILURE_PLACEHOLDER}) #{r}" }.join
      output_row(type: :dump_failures, formatted_output: formatted) unless formatted.empty?

      output_row(
        type: :summary,
        count: count,
        assertions: assertions,
        failures: failures,
        errors: errors,
        skips: skips
      )
    end

    private

    # Minitest::Result (5.11+) has class_name; older minitest hands the
    # reporter the test instance itself.
    def class_name_of(result)
      result.respond_to?(:class_name) ? result.class_name : result.class.name
    end

    def output_row(obj)
      io.puts SEPARATOR + obj.to_json
      io.flush
    end
  end
end

Minitest.extensions << "plur" unless Minitest.extensions.include?("plur")

# Registered via Minitest.extensions above, so this runs on minitest 5 and 6
# alike - unlike *_plugin.rb autoloading, which minitest 6 made opt-in. The
# takeover happens in CompositeReporter#start, which Minitest.run calls after
# every plugin init, so it wins regardless of load order against plugins that
# rewrite the composite (minitest-reporters, Rails).
def Minitest.plugin_plur_init(options)
  plur_io = options[:io]
  Minitest::CompositeReporter.prepend(Module.new do
    define_method(:start) do
      unless reporters.any? { |r| Plur::MinitestReporter === r }
        reporters.clear
        reporters << Plur::MinitestReporter.new(plur_io, options)
      end
      super()
    end
  end)
end
