require "spec_helper"

RSpec.describe "Backspin system() support" do
  describe "recording system calls" do
    it "records a simple system call" do
      result = Backspin.call("system_echo") do
        system("echo hello")
      end

      expect(result.commands.size).to eq(1)
      command = result.commands.first
      expect(command.args).to eq(["echo", "hello"])
      expect(command.stdout).to eq("hello\n")
      expect(command.stderr).to eq("")
      expect(command.status).to eq(0)
    end

    it "records system call exit status" do
      result = Backspin.call("system_false") do
        system("false")
      end

      command = result.commands.first
      expect(command.status).to eq(1)
    end

    it "records multiple system calls" do
      result = Backspin.call("multi_system") do
        system("echo first")
        system("echo second")
      end

      expect(result.commands.size).to eq(2)
      expect(result.commands[0].stdout).to eq("first\n")
      expect(result.commands[1].stdout).to eq("second\n")
    end

    it "captures stderr from system calls" do
      result = Backspin.call("system_stderr") do
        system("echo error >&2")
      end

      command = result.commands.first
      expect(command.stdout).to eq("")
      expect(command.stderr).to eq("error\n")
    end
  end

  describe "verifying system calls" do
    it "verifies matching system output" do
      # Record
      Backspin.call("verify_system") do
        system("echo hello")
      end

      # Verify - should pass
      result = Backspin.verify("verify_system") do
        system("echo hello")
      end

      expect(result.verified?).to be true
    end

    it "detects non-matching system output" do
      # Record
      Backspin.call("verify_system_diff") do
        system("echo original")
      end

      # Verify - should fail
      result = Backspin.verify("verify_system_diff") do
        system("echo different")
      end

      expect(result.verified?).to be false
    end
  end

  describe "playback mode" do
    it "plays back system calls without executing" do
      # Record
      Backspin.call("playback_system") do
        system("echo recorded")
      end

      # Playback
      result = Backspin.verify("playback_system", mode: :playback) do
        # This should not actually execute
        system("echo not executed")
      end

      expect(result.verified?).to be true
      expect(result.command_executed?).to be false
    end
  end

  describe "mixing system and capture3" do
    it "records both system and capture3 calls" do
      result = Backspin.call("mixed_calls") do
        system("echo from system")
        Open3.capture3("echo from capture3")
      end

      expect(result.commands.size).to eq(2)
      expect(result.commands[0].class.name).to eq("Kernel::System")
      expect(result.commands[0].stdout).to eq("from system\n")
      expect(result.commands[1].class.name).to eq("Open3::Capture3")
      expect(result.commands[1].stdout).to eq("from capture3\n")
    end
  end
end
