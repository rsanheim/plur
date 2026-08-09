# frozen_string_literal: true

# Plur's minitest plugin, written to $PLUR_HOME/config/ruby/minitest/ and
# discovered by minitest's own plugin loading (Gem.find_files searches the
# $LOAD_PATH plur extends with -I). Minitest requires this file itself, so
# Minitest is already fully loaded here and nothing needs requiring early.
#
# On minitest 5.x discovery happens automatically inside Minitest.run; on
# minitest 6 plugin loading is opt-in, so plur's worker script calls the
# documented Minitest.load_plugins after the test files are required.
require "json"

module Plur
  # Streams one JSON row per test to minitest's own output IO (options[:io],
  # captured before any test code can swap $stdout). With ProgressReporter
  # and SummaryReporter removed from the composite, any bare line on stdout
  # is test-written output by elimination.
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

# The documented plugin hook (README.rdoc "Writing Extensions"): minitest
# calls this during init_plugins with Minitest.reporter - the top-level
# composite - available. Remove minitest's two default output reporters and
# add ours alongside anything other plugins installed. The composite itself
# is never replaced or patched, so composites that test code constructs
# (meta-testing suites) are untouched.
#
# Known trades: a plugin whose init runs after ours and rewrites the
# composite wholesale (minitest-reporters) wins; and with SummaryReporter
# gone, minitest's empty_run! guard never fires, so a filter matching
# nothing exits 0 instead of 1 (same deviation minitest-reporters has).
def Minitest.plugin_plur_init(options)
  reporter = Minitest.reporter
  return unless reporter
  return if reporter.reporters.any? { |r| Plur::MinitestReporter === r }

  reporter.reporters.reject! { |r| Minitest::ProgressReporter === r || Minitest::SummaryReporter === r }
  reporter << Plur::MinitestReporter.new(options[:io], options)
end
