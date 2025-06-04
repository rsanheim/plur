require "spec_helper"

RSpec.describe Backspin::Dubplate do
  let(:dubplate_path) { "tmp/backspin/test_dubplate.yaml" }
  let(:dubplate) { described_class.new(dubplate_path) }

  before do
    FileUtils.rm_rf("tmp/backspin")
  end

  after do
    FileUtils.rm_rf("tmp/backspin")
  end

  describe "#initialize" do
    it "creates a new dubplate with the given path" do
      expect(dubplate.path).to eq(dubplate_path)
    end

    it "initializes with empty commands if file doesn't exist" do
      expect(dubplate.commands).to eq([])
    end

    it "loads existing commands if file exists" do
      # Create a dubplate file with some data
      FileUtils.mkdir_p(File.dirname(dubplate_path))
      File.write(dubplate_path, [
        {"stdout" => "hello\n", "stderr" => "", "status" => 0, "recorded_at" => "2024-01-01T00:00:00Z"}
      ].to_yaml)

      loaded_dubplate = described_class.new(dubplate_path)
      expect(loaded_dubplate.commands.size).to eq(1)
      expect(loaded_dubplate.commands.first).to be_a(Backspin::Command)
      expect(loaded_dubplate.commands.first.stdout).to eq("hello\n")
    end

    it "raises error if file exists but has invalid format" do
      FileUtils.mkdir_p(File.dirname(dubplate_path))
      File.write(dubplate_path, "not an array")

      expect {
        described_class.new(dubplate_path)
      }.to raise_error(Backspin::DubplateFormatError, /Invalid dubplate format/)
    end
  end

  describe "#add_command" do
    it "adds a command to the dubplate" do
      command = Backspin::Command.new(
        method_class: Open3::Capture3,
        args: ["echo", "hello"],
        stdout: "hello\n",
        stderr: "",
        status: 0,
        recorded_at: Time.now.iso8601
      )

      dubplate.add_command(command)
      expect(dubplate.commands).to include(command)
    end

    it "returns self for chaining" do
      command = Backspin::Command.new(
        method_class: Open3::Capture3,
        args: ["echo", "test"],
        stdout: "test\n",
        stderr: "",
        status: 0,
        recorded_at: Time.now.iso8601
      )

      expect(dubplate.add_command(command)).to eq(dubplate)
    end
  end

  describe "#save" do
    it "creates the directory if it doesn't exist" do
      dubplate.add_command(create_test_command("test"))
      dubplate.save

      expect(File.directory?(File.dirname(dubplate_path))).to be true
    end

    it "writes commands to YAML file" do
      dubplate.add_command(create_test_command("first"))
      dubplate.add_command(create_test_command("second"))
      dubplate.save

      expect(File.exist?(dubplate_path)).to be true

      saved_data = YAML.load_file(dubplate_path)
      expect(saved_data).to be_an(Array)
      expect(saved_data.size).to eq(2)
      expect(saved_data[0]["stdout"]).to eq("first\n")
      expect(saved_data[1]["stdout"]).to eq("second\n")
    end

    it "overwrites existing file" do
      # Create initial file
      FileUtils.mkdir_p(File.dirname(dubplate_path))
      File.write(dubplate_path, [{"stdout" => "old\n"}].to_yaml)

      # Create a new dubplate instance that won't load the existing data
      new_dubplate = described_class.new(dubplate_path)
      expect(new_dubplate.size).to eq(1)  # It loaded the existing data

      # Clear and add new data
      new_dubplate.clear
      new_dubplate.add_command(create_test_command("new"))
      new_dubplate.save

      saved_data = YAML.load_file(dubplate_path)
      expect(saved_data.size).to eq(1)
      expect(saved_data[0]["stdout"]).to eq("new\n")
    end

    it "applies credential scrubbing when saving" do
      command = Backspin::Command.new(
        method_class: Open3::Capture3,
        args: ["aws"],
        stdout: "AKIA1234567890123456\n",
        stderr: "",
        status: 0,
        recorded_at: Time.now.iso8601
      )

      dubplate.add_command(command)
      dubplate.save

      saved_data = YAML.load_file(dubplate_path)
      expect(saved_data[0]["stdout"]).to eq("********************\n")
    end
  end

  describe "#reload" do
    it "reloads commands from disk" do
      # Start with one command
      dubplate.add_command(create_test_command("first"))
      dubplate.save

      # Modify the file externally
      new_data = [
        {"stdout" => "external\n", "stderr" => "", "status" => 0, "recorded_at" => Time.now.iso8601}
      ]
      File.write(dubplate_path, new_data.to_yaml)

      # Reload
      dubplate.reload

      expect(dubplate.commands.size).to eq(1)
      expect(dubplate.commands.first.stdout).to eq("external\n")
    end

    it "clears commands if file doesn't exist" do
      dubplate.add_command(create_test_command("test"))
      FileUtils.rm_rf(File.dirname(dubplate_path))

      dubplate.reload

      expect(dubplate.commands).to eq([])
    end
  end

  describe "#exists?" do
    it "returns false when file doesn't exist" do
      expect(dubplate.exists?).to be false
    end

    it "returns true when file exists" do
      dubplate.save
      expect(dubplate.exists?).to be true
    end
  end

  describe "#empty?" do
    it "returns true when no commands are recorded" do
      expect(dubplate.empty?).to be true
    end

    it "returns false when commands are present" do
      dubplate.add_command(create_test_command("test"))
      expect(dubplate.empty?).to be false
    end
  end

  describe "#size" do
    it "returns the number of recorded commands" do
      expect(dubplate.size).to eq(0)

      dubplate.add_command(create_test_command("first"))
      expect(dubplate.size).to eq(1)

      dubplate.add_command(create_test_command("second"))
      expect(dubplate.size).to eq(2)
    end
  end

  describe "#next_command" do
    it "returns commands in sequence" do
      dubplate.add_command(create_test_command("first"))
      dubplate.add_command(create_test_command("second"))

      expect(dubplate.next_command.stdout).to eq("first\n")
      expect(dubplate.next_command.stdout).to eq("second\n")
    end

    it "raises error when no more commands are available" do
      dubplate.add_command(create_test_command("only"))

      dubplate.next_command # Use the only command

      expect {
        dubplate.next_command
      }.to raise_error(Backspin::NoMoreRecordingsError, /No more recordings available/)
    end

    it "resets playback position when reloaded" do
      dubplate.add_command(create_test_command("test"))
      dubplate.save  # Save to disk first
      dubplate.next_command # Use the command

      dubplate.reload # This should reset position
      expect(dubplate.next_command.stdout).to eq("test\n")
    end
  end

  describe "#clear" do
    it "removes all commands" do
      dubplate.add_command(create_test_command("first"))
      dubplate.add_command(create_test_command("second"))

      dubplate.clear

      expect(dubplate.empty?).to be true
      expect(dubplate.size).to eq(0)
    end
  end

  describe ".load_or_create" do
    it "loads existing dubplate" do
      # Create a dubplate file
      FileUtils.mkdir_p(File.dirname(dubplate_path))
      File.write(dubplate_path, [
        {"stdout" => "existing\n", "stderr" => "", "status" => 0, "recorded_at" => "2024-01-01T00:00:00Z"}
      ].to_yaml)

      dubplate = described_class.load_or_create(dubplate_path)

      expect(dubplate.size).to eq(1)
      expect(dubplate.commands.first.stdout).to eq("existing\n")
    end

    it "creates new dubplate if file doesn't exist" do
      dubplate = described_class.load_or_create(dubplate_path)

      expect(dubplate.empty?).to be true
      expect(dubplate.path).to eq(dubplate_path)
    end
  end

  private

  def create_test_command(output)
    Backspin::Command.new(
      method_class: Open3::Capture3,
      args: ["echo", output],
      stdout: "#{output}\n",
      stderr: "",
      status: 0,
      recorded_at: Time.now.iso8601
    )
  end
end
