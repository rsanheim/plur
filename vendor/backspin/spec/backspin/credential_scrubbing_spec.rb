require "spec_helper"

RSpec.describe "Backspin credential scrubbing" do
  let(:backspin_path) { Pathname.new(File.join("tmp", "backspin")) }

  describe "configuration" do
    it "has credential scrubbing enabled by default" do
      expect(Backspin.configuration.scrub_credentials).to be true
    end

    it "can disable credential scrubbing" do
      Backspin.configure do |config|
        config.scrub_credentials = false
      end

      expect(Backspin.configuration.scrub_credentials).to be false

      # Reset for other tests
      Backspin.reset_configuration!
    end

    it "can add custom credential patterns" do
      Backspin.configure do |config|
        config.add_credential_pattern(/MY_SECRET_[A-Z0-9]+/)
      end

      expect(Backspin.configuration.credential_patterns).to include(/MY_SECRET_[A-Z0-9]+/)

      # Reset for other tests
      Backspin.reset_configuration!
    end
  end

  describe "scrubbing AWS credentials" do
    it "scrubs AWS access key IDs" do
      result = Backspin.record("aws_keys") do
        Open3.capture3("echo AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      expect(dubplate_data.first["stdout"]).to eq("AWS_ACCESS_KEY_ID=********************\n")
    end

    it "scrubs AWS secret keys" do
      result = Backspin.record("aws_secret") do
        Open3.capture3("echo aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      # The entire match including key name gets scrubbed
      expect(dubplate_data.first["stdout"]).to eq("#{"*" * 62}\n")
    end
  end

  describe "scrubbing Google credentials" do
    it "scrubs Google API keys" do
      result = Backspin.record("google_api_key") do
        Open3.capture3("echo GOOGLE_API_KEY=AIzaSyDaGmWKa4JsXZ-HjGw7ISLn_3namBGewQe")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      expect(dubplate_data.first["stdout"]).to eq("GOOGLE_API_KEY=***************************************\n")
    end
  end

  describe "scrubbing generic credentials" do
    it "scrubs API keys" do
      result = Backspin.record("api_key") do
        Open3.capture3("echo api_key=abc123def456ghi789jkl012mno345pqr678")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      # The entire match including key name gets scrubbed
      expect(dubplate_data.first["stdout"]).to eq("#{"*" * 44}\n")
    end

    it "scrubs passwords" do
      result = Backspin.record("password") do
        Open3.capture3("echo 'database password: supersecretpassword123!'")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      # The pattern matches "password: supersecretpassword123!"
      expect(dubplate_data.first["stdout"]).to eq("database #{"*" * 33}\n")
    end
  end

  describe "scrubbing stderr" do
    it "scrubs credentials from stderr as well" do
      result = Backspin.record("stderr_creds") do
        Open3.capture3("sh -c 'echo normal output && echo \"Error: Invalid API_KEY=sk-1234567890abcdef1234567890abcdef\" >&2 && exit 1'")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      expect(dubplate_data.first["stdout"]).to eq("normal output\n")
      # The entire match including key name gets scrubbed
      expect(dubplate_data.first["stderr"]).to eq("Error: Invalid #{"*" * 43}\n")
    end
  end

  describe "when scrubbing is disabled" do
    before do
      Backspin.configure do |config|
        config.scrub_credentials = false
      end
    end

    after do
      Backspin.reset_configuration!
    end

    it "does not scrub credentials" do
      result = Backspin.record("no_scrub") do
        Open3.capture3("echo AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      expect(dubplate_data.first["stdout"]).to eq("AWS_ACCESS_KEY_ID=AKIAIOSFODNN7EXAMPLE\n")
    end
  end

  describe "private key detection" do
    it "scrubs private keys" do
      result = Backspin.record("private_key") do
        Open3.capture3("echo '-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANB...'")
      end

      dubplate_data = YAML.load_file(result.dubplate_path)
      expect(dubplate_data.first["stdout"]).to match(/\*{27}/)
    end
  end
end
