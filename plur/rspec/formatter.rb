# frozen_string_literal: true

require "json"

module Plur
  # A streaming JSON formatter for plur that outputs one JSON object per line
  # Based on TurboTests::JsonRowsFormatter but simplified for plur usage
  class JsonRowsFormatter
    attr_reader :output
    RSpec::Core::Formatters.register(
      self,
      :start,
      :close,
      :example_failed,
      :example_passed,
      :example_pending,
      :example_group_started,
      :example_group_finished,
      :message,
      :seed,
      :dump_failures,
      :dump_summary
    )

    def initialize(output)
      @output = output
      @separator = ENV["PLUR_FORMATTER_SEPARATOR"] || "PLUR_JSON:"
    end

    def start(notification)
      output_row(
        type: :load_summary,
        summary: {
          count: notification.count,
          load_time: notification.load_time
        }
      )
    end

    def example_group_started(notification)
      output_row(
        type: :group_started,
        group: {
          description: notification.group.description,
          file_path: notification.group.file_path,
          line_number: notification.group.location.split(":").last.to_i
        }
      )
    end

    def example_group_finished(notification)
      output_row(
        type: :group_finished,
        group: {
          description: notification.group.description
        }
      )
    end

    def example_passed(notification)
      output_row(
        type: :example_passed,
        example: example_to_json(notification.example)
      )
    end

    def example_pending(notification)
      output_row(
        type: :example_pending,
        example: example_to_json(notification.example)
      )
    end

    def example_failed(notification)
      output_row(
        type: :example_failed,
        example: example_to_json(notification.example)
      )
    end

    def seed(notification)
      output_row(
        type: :seed,
        seed: notification.seed
      )
    end

    def close(notification)
      output_row(
        type: :close
      )
    end

    def message(notification)
      output_row(
        type: :message,
        message: notification.message
      )
    end

    def dump_failures(notification)
      # Don't output anything if there are no failures
      return if notification.failure_notifications.empty?

      # Capture the fully formatted failures with colors
      output_row(
        type: :dump_failures,
        formatted_output: notification.fully_formatted_failed_examples
      )
    end

    def dump_summary(summary)
      # Capture the fully formatted summary with colors
      output_row(
        type: :dump_summary,
        formatted_output: summary.fully_formatted,
        totals_line: summary.totals_line,
        colorized_totals_line: summary.colorized_totals_line,
        example_count: summary.example_count,
        failure_count: summary.failure_count,
        pending_count: summary.pending_count,
        duration: summary.duration,
        load_time: summary.load_time
      )
    end

    private

    def example_to_json(example)
      {
        description: example.description,
        full_description: example.full_description,
        location: example.location,
        file_path: example.file_path,
        line_number: example.location.split(":").last.to_i,
        status: example.execution_result.status,
        run_time: example.execution_result.run_time,
        pending_message: example.execution_result.pending_message,
        exception: exception_to_json(example.execution_result.exception)
      }
    end

    def exception_to_json(exception)
      return nil unless exception

      {
        class: exception.class.name,
        message: exception.message,
        backtrace: format_backtrace(exception.backtrace)
      }
    end

    # Use RSpec's built-in backtrace filtering for filtering, though we have a rescue 
    # here to handle the case where RSpec's backtrace formatter fails.
    # This can happen if the test suite is changing the cwd in around hook for parallel runs.
    # This is done in Rspec suite's for example: 
    # https://github.com/rspec/rspec/blob/0c37a88d4ff511debd563d68e10c1c7672318c3c/rspec-support/lib/rspec/support/spec/with_isolated_directory.rb#L5-L15
    #
    # Returns the backtrace as an array of strings
    def format_backtrace(backtrace)
      return [] unless backtrace

      RSpec.configuration.backtrace_formatter.format_backtrace(backtrace)
    rescue Errno::ENOENT
      backtrace
    end

    def output_row(obj)
      output.puts @separator + obj.to_json
      output.flush
    end
  end
end
