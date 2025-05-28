require 'rspec'
require 'json'
require_relative '../lib/rux/json_rows_formatter'

RSpec.describe Rux::JsonRowsFormatter do
  let(:output) { StringIO.new }
  let(:formatter) { described_class.new(output) }
  
  def json_messages
    output.string.lines.map do |line|
      separator = ENV["RUX_FORMATTER_SEPARATOR"] || "RUX_JSON:"
      next unless line.start_with?(separator)
      JSON.parse(line.sub(separator, ''))
    end.compact
  end

  describe '#start' do
    it 'outputs a start message with count and load time' do
      notification = double('notification', count: 5, load_time: 0.123)
      
      formatter.start(notification)
      
      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "start",
        "count" => 5,
        "load_time" => 0.123
      })
    end
  end

  describe '#example_passed' do
    it 'outputs a passed example message' do
      example = double('example',
        description: 'does something',
        full_description: 'MyClass does something',
        location: './spec/foo_spec.rb:10',
        file_path: './spec/foo_spec.rb',
        execution_result: double('result',
          status: :passed,
          run_time: 0.01,
          pending_message: nil,
          exception: nil
        )
      )
      notification = double('notification', example: example)
      
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

  describe '#example_failed' do
    it 'outputs a failed example message with exception details' do
      exception = StandardError.new("Something went wrong")
      exception.set_backtrace([
        "/path/to/file.rb:123:in `method'",
        "/path/to/another.rb:456:in `block'"
      ])
      
      example = double('example',
        description: 'fails',
        full_description: 'MyClass fails',
        location: './spec/foo_spec.rb:20',
        file_path: './spec/foo_spec.rb',
        execution_result: double('result',
          status: :failed,
          run_time: 0.02,
          pending_message: nil,
          exception: exception
        )
      )
      notification = double('notification', example: example)
      
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

  describe '#example_group_started' do
    it 'outputs a group started message' do
      group = double('group',
        description: 'MyClass',
        file_path: './spec/my_class_spec.rb',
        location: './spec/my_class_spec.rb:5'
      )
      notification = double('notification', group: group)
      
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

  describe '#seed' do
    it 'outputs a seed message' do
      notification = double('notification', seed: 12345)
      
      formatter.seed(notification)
      
      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "seed",
        "seed" => 12345
      })
    end
  end

  describe '#close' do
    it 'outputs a close message' do
      notification = double('notification')
      
      formatter.close(notification)
      
      messages = json_messages
      expect(messages.size).to eq(1)
      expect(messages[0]).to eq({
        "type" => "close"
      })
    end
  end

  describe 'output format' do
    it 'prefixes each JSON line with the separator' do
      notification = double('notification', seed: 42)
      
      formatter.seed(notification)
      
      lines = output.string.lines
      expect(lines.size).to eq(1)
      expect(lines[0]).to start_with("RUX_JSON:")
      expect(lines[0]).to include('"type":"seed"')
    end

    it 'respects custom separator from environment variable' do
      ENV['RUX_FORMATTER_SEPARATOR'] = 'CUSTOM:'
      
      notification = double('notification', seed: 42)
      formatter.seed(notification)
      
      lines = output.string.lines
      expect(lines[0]).to start_with("CUSTOM:")
      
      ENV.delete('RUX_FORMATTER_SEPARATOR')
    end
  end
end