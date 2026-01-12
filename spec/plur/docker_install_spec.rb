require "spec_helper"
require "plur/docker_install"

RSpec.describe Plur::DockerInstall do
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
      architectures = Plur::DockerInstall::ARCHITECTURES
      expect(architectures["x86_64"]).to eq("amd64")
      expect(architectures["aarch64"]).to eq("arm64")
      expect(architectures["arm64"]).to eq("arm64")
    end
  end
end
