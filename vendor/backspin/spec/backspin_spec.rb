require "spec_helper"

RSpec.describe Backspin do
  let(:backspin_path) { Pathname.new(File.join("tmp", "backspin")) }

  context "record" do
    it "records stdout, stderr, and status to a single yaml" do
      time = Time.now
      Timecop.freeze(time)
      result = Backspin.record("echo_hello") do
        Open3.capture3("echo hello")
      end
      expect(result.commands.size).to eq(1)
      expect(result.commands.first.class).to eq(Open3::Capture3)
      expect(result.commands.first.args).to eq(["echo", "hello"])
      expect(result.dubplate_path.to_s).to end_with("tmp/backspin/echo_hello.yaml")
      expect(backspin_path.join("echo_hello.yaml")).to exist
      results = YAML.load_file(backspin_path.join("echo_hello.yaml"))
      expect(results).to be_a(Array)
      expect(results.size).to eq(1)
      expect(results.first).to eq({
        "stdout" => "hello\n",
        "stderr" => "",
        "status" => 0,
        "recorded_at" => time.iso8601
      })

      expect(Backspin.output).to eq("hello\n")
    end

    it "raises ArgumentError when no dubplate name is provided" do
      expect {
        Backspin.record do
          Open3.capture3("echo hello")
        end
      }.to raise_error(ArgumentError, "wrong number of arguments (given 0, expected 1)")
    end

    it "raises ArgumentError when dubplate name is nil" do
      expect {
        Backspin.record(nil) do
          Open3.capture3("echo hello")
        end
      }.to raise_error(ArgumentError, "dubplate_name is required")
    end

    it "raises ArgumentError when dubplate name is empty" do
      expect {
        Backspin.record("") do
          Open3.capture3("echo hello")
        end
      }.to raise_error(ArgumentError, "dubplate_name is required")
    end

    it "records multiple commands to an array in the dubplate" do
      time = Time.now
      Timecop.freeze(time)

      result = Backspin.record("multi_command") do
        Open3.capture3("echo first")
        Open3.capture3("echo second")
        Open3.capture3("echo third")
      end

      expect(result.commands.size).to eq(3)
      expect(result.commands[0].args).to eq(["echo", "first"])
      expect(result.commands[1].args).to eq(["echo", "second"])
      expect(result.commands[2].args).to eq(["echo", "third"])

      expect(result.dubplate_path).to exist
      dubplate_data = YAML.load_file(result.dubplate_path)

      # Multiple commands should be stored as an array
      expect(dubplate_data).to be_a(Array)
      expect(dubplate_data.size).to eq(3)

      expect(dubplate_data[0]).to eq({
        "stdout" => "first\n",
        "stderr" => "",
        "status" => 0,
        "recorded_at" => time.iso8601
      })

      expect(dubplate_data[1]).to eq({
        "stdout" => "second\n",
        "stderr" => "",
        "status" => 0,
        "recorded_at" => time.iso8601
      })

      expect(dubplate_data[2]).to eq({
        "stdout" => "third\n",
        "stderr" => "",
        "status" => 0,
        "recorded_at" => time.iso8601
      })

      # last_output should be the last command
      expect(Backspin.output).to eq("third\n")
    end
  end
end
