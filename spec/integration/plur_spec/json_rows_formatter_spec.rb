require "rspec"
require "json"
require_relative "../../../plur/rspec/formatter"

RSpec.describe Plur::JsonRowsFormatter do
  let(:output) { StringIO.new }
  let(:formatter) { described_class.new(output) }

  def json_messages
    output.string.lines.map do |line|
      separator = ENV["PLUR_FORMATTER_SEPARATOR"] || "PLUR_JSON:"
      next unless line.start_with?(separator)
      JSON.parse(line.sub(separator, ""))
    end.compact
  end

  describe "#start" do
    it "outputs a load_summary message with count and load time" do
      notification = double("notification", count: 5, load_time: 0.123)

      formatter.start(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "load_summary",
        "summary" => {
          "count" => 5,
          "load_time" => 0.123
        }
      })
    end
  end

  describe "#example_passed" do
    it "outputs a passed example message" do
      example = double("example",
        description: "does something",
        full_description: "MyClass does something",
        location: "./spec/foo_spec.rb:10",
        file_path: "./spec/foo_spec.rb",
        execution_result: double("result",
          status: :passed,
          run_time: 0.01,
          pending_message: nil,
          exception: nil))
      notification = double("notification", example: example)

      formatter.example_passed(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to include({
        "type" => "example_passed",
        "example" => {
          "description" => "does something",
          "full_description" => "MyClass does something",
          "location" => "./spec/foo_spec.rb:10",
          "file_path" => "./spec/foo_spec.rb",
          "line_number" => 10,
          "status" => "passed",
          "run_time" => 0.01,
          "pending_message" => nil,
          "exception" => nil
        }
      })
    end
  end

  describe "#example_failed" do
    it "outputs a failed example message with exception details" do
      exception = StandardError.new("Something went wrong")
      exception.set_backtrace([
        "/path/to/file.rb:123:in `method'",
        "/path/to/another.rb:456:in `block'"
      ])

      example = double("example",
        description: "fails",
        full_description: "MyClass fails",
        location: "./spec/foo_spec.rb:20",
        file_path: "./spec/foo_spec.rb",
        execution_result: double("result",
          status: :failed,
          run_time: 0.02,
          pending_message: nil,
          exception: exception))
      notification = double("notification", example: example)

      # Mock RSpec configuration for backtrace formatting
      config_double = double("config", dry_run?: false)
      formatter_double = double("backtrace_formatter")
      allow(formatter_double).to receive(:format_backtrace).with(anything) { |backtrace| backtrace }
      allow(config_double).to receive(:backtrace_formatter).and_return(formatter_double)
      allow(RSpec).to receive(:configuration).and_return(config_double)

      formatter.example_failed(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]["type"]).to eq("example_failed")
      expect(messages[0]["example"]["status"]).to eq("failed")
      expect(messages[0]["example"]["exception"]).to eq({
        "class" => "StandardError",
        "message" => "Something went wrong",
        "backtrace" => [
          "/path/to/file.rb:123:in `method'",
          "/path/to/another.rb:456:in `block'"
        ]
      })
    end
  end

  describe "#example_group_started" do
    it "outputs a group started message" do
      group = double("group",
        description: "MyClass",
        file_path: "./spec/my_class_spec.rb",
        location: "./spec/my_class_spec.rb:5")
      notification = double("notification", group: group)

      formatter.example_group_started(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "group_started",
        "group" => {
          "description" => "MyClass",
          "file_path" => "./spec/my_class_spec.rb",
          "line_number" => 5
        }
      })
    end
  end

  describe "#seed" do
    it "outputs a seed message" do
      notification = double("notification", seed: 12345)

      formatter.seed(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "seed",
        "seed" => 12345
      })
    end
  end

  describe "#close" do
    it "outputs a close message" do
      notification = double("notification")

      formatter.close(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "close"
      })
    end
  end

  describe "#dump_failures" do
    # We are using real rspec objects as much as possible to try and catch any regressions
    # in in between rspec versions.
    it "outputs a dump_failures message when there are failures" do
      error = nil
      # force a failure and catch it to use in our test
      begin
        expect(2).to eq(3)
      rescue RSpec::Expectations::ExpectationNotMetError => e
        error = e
      end
      result = RSpec::Core::Example::ExecutionResult.new
      result.started_at = Time.now
      result.record_finished(:failed, Time.now)
      result.exception = error

      example_group = class_double(RSpec::Core::ExampleGroup, description: "TestGroup")
      allow(example_group).to receive(:parent_groups).and_return([example_group])

      example = instance_double(
        RSpec::Core::Example,
        description: "Example failed",
        full_description: "TestGroup Example failed",
        example_group: example_group,
        execution_result: result,
        location: "./spec/example_spec.rb:10",
        location_rerun_argument: "./spec/example_spec.rb:10",
        metadata: {shared_group_inclusion_backtrace: []}
      )

      failure_notification = RSpec::Core::Notifications::ExampleNotification.for(example)
      notification = Struct.new(:failure_notifications).new([failure_notification])

      formatter.dump_failures(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]["type"]).to eq("dump_failures")
      expect(messages[0]["formatted_output"]).to include(described_class::FAILURE_PLACEHOLDER + ")")
      expect(messages[0]["formatted_output"]).to include("Example failed")
      expect(messages[0]["formatted_output"]).to include(File.basename(__FILE__))
    end

    it "outputs nothing when there are no failures" do
      notification = double("notification", failure_notifications: [])

      formatter.dump_failures(notification)

      messages = json_messages
      expect(messages).to be_empty
    end
  end

  describe "#dump_pending" do
    it "outputs a dump_pending message when there are pending specs" do
      # Following RSpec's own test pattern: real ExecutionResult + instance_double Example
      result = RSpec::Core::Example::ExecutionResult.new
      result.started_at = Time.now
      result.record_finished(:pending, Time.now)
      result.pending_message = "Not yet implemented"

      example_group = class_double(RSpec::Core::ExampleGroup, description: "TestGroup")
      allow(example_group).to receive(:parent_groups).and_return([example_group])

      example = instance_double(
        RSpec::Core::Example,
        description: "Example pending",
        full_description: "TestGroup Example pending",
        example_group: example_group,
        execution_result: result,
        location: "./spec/example_spec.rb:20",
        location_rerun_argument: "./spec/example_spec.rb:20",
        metadata: {shared_group_inclusion_backtrace: []}
      )

      pending_notification = RSpec::Core::Notifications::ExampleNotification.for(example)
      notification = Struct.new(:pending_notifications).new([pending_notification])

      formatter.dump_pending(notification)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]["type"]).to eq("dump_pending")
      expect(messages[0]["formatted_output"]).to include(Plur::JsonRowsFormatter::PENDING_PLACEHOLDER + ")")
      expect(messages[0]["formatted_output"]).to include("Example pending")
    end

    it "outputs nothing when there are no pending specs" do
      notification = double("notification", pending_notifications: [])

      formatter.dump_pending(notification)

      messages = json_messages
      expect(messages).to be_empty
    end
  end

  describe "#dump_summary" do
    it "outputs a dump_summary message" do
      summary = double("summary",
        fully_formatted: "Finished in 1.23 seconds\n10 examples, 0 failures",
        totals_line: "10 examples, 0 failures",
        colorized_totals_line: "\e[32m10 examples, 0 failures\e[0m",
        example_count: 10,
        failure_count: 0,
        pending_count: 0,
        duration: 1.23,
        load_time: 0.1)

      formatter.dump_summary(summary)

      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "dump_summary",
        "formatted_output" => "Finished in 1.23 seconds\n10 examples, 0 failures",
        "totals_line" => "10 examples, 0 failures",
        "colorized_totals_line" => "\e[32m10 examples, 0 failures\e[0m",
        "example_count" => 10,
        "failure_count" => 0,
        "pending_count" => 0,
        "duration" => 1.23,
        "load_time" => 0.1
      })
    end
  end

  describe "output format" do
    it "prefixes each JSON line with the separator" do
      notification = double("notification", seed: 42)

      formatter.seed(notification)

      lines = output.string.lines
      expect(lines.size).to eq(1)
      expect(lines[0]).to start_with("PLUR_JSON:")
      expect(lines[0]).to include('"type":"seed"')
    end

    it "respects custom separator from environment variable" do
      ENV["PLUR_FORMATTER_SEPARATOR"] = "CUSTOM:"

      notification = double("notification", seed: 42)
      formatter.seed(notification)

      lines = output.string.lines
      expect(lines[0]).to start_with("CUSTOM:")

      ENV.delete("PLUR_FORMATTER_SEPARATOR")
    end
  end
end
