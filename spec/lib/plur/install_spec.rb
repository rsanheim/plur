require "spec_helper"
require "plur/install"

RSpec.describe Plur::Install do
  let(:install) { described_class.new }

  describe "#run" do
    context "when container does not exist" do
      it "exits with an error message" do
        expect {
          install.run(command: "install", service: "non-existent-container")
        }.to raise_error(SystemExit)
          .and output(/Error: Could not find running container: non-existent-container/).to_stdout
      end
    end

    # For containers that do exist, we'd need real Docker containers
    # or integration tests. For unit tests, we can test the key behaviors
    # by testing the individual methods that do the actual work
  end

  describe "#build_container_name" do
    it "returns service name without prefix" do
      expect(install.send(:build_container_name, "my-app", nil)).to eq("my-app")
    end

    it "builds compose container name with prefix" do
      expect(install.send(:build_container_name, "web", "myproject")).to eq("myproject_web_1")
    end
  end

  describe "#detect_architecture" do
    it "maps common architectures correctly" do
      # This would require a running container to test properly
      # We can test the mapping logic separately
      architectures = Plur::Install::ARCHITECTURES
      expect(architectures["x86_64"]).to eq("amd64")
      expect(architectures["aarch64"]).to eq("arm64")
      expect(architectures["arm64"]).to eq("arm64")
    end
  end

  describe "#find_or_build_binary" do
    let(:dist_dir) { Plur.config.root_dir.join("dist") }

    context "when binary exists" do
      it "returns the binary path" do
        # Create a temporary binary file for testing
        FileUtils.mkdir_p(dist_dir)
        binary_path = dist_dir.join("plur-linux-arm64")
        FileUtils.touch(binary_path)

        result = install.send(:find_or_build_binary, "arm64")
        expect(result).to eq(binary_path)

        # Cleanup
        FileUtils.rm_f(binary_path)
      end
    end

    context "when binary does not exist" do
      it "builds the binary" do
        # This is hard to test without actually running the build
        # In a real test suite, this would be an integration test
        skip "Requires integration test setup"
      end
    end
  end

  # Integration-style test that actually exercises the real commands
  describe "integration" do
    context "with a real Docker container", skip: "requires Docker" do
      before do
        # This would set up a test container
        # system("docker run -d --name test-plur-container alpine:latest sleep 3600")
      end

      after do
        # system("docker rm -f test-plur-container")
      end

      it "can check container status" do
        # install.run(command: "status", service: "test-plur-container")
      end
    end
  end
end
