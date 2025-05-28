# frozen_string_literal: true

require "json"
require "rspec/core"
require "rspec/core/formatters"
require "rspec/core/notifications"

module Rux
  # A streaming JSON formatter for rux that outputs one JSON object per line
  # Based on TurboTests::JsonRowsFormatter but simplified for rux usage
  class JsonRowsFormatter
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
      :seed
    )

    attr_reader :output

    def initialize(output)
      @output = output
    end

    def start(notification)
      output_row(
        type: :start,
        count: notification.count,
        load_time: notification.load_time
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

    def format_backtrace(backtrace)
      return [] unless backtrace
      
      # Use RSpec's built-in backtrace filtering
      RSpec.configuration.backtrace_formatter.format_backtrace(backtrace)
    end

    def output_row(obj)
      # Use environment variable for separator, default to "RUX_JSON:"
      separator = ENV["RUX_FORMATTER_SEPARATOR"] || "RUX_JSON:"
      output.puts separator + obj.to_json
      output.flush
    end
  end
end