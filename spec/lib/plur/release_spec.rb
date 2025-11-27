require "spec_helper"
require "plur/release"

RSpec.describe "script/release" do
  describe "help" do
    it "shows usage with current version" do
      output = `script/release --help`

      expect(output).to include("Usage: script/release <command> <version>")
      expect(output).to include("Current version:")
      expect(output).to include("prepare VERSION")
      expect(output).to include("push VERSION")
      expect(output).to include("extract-notes VERSION")
    end
  end

  describe "extract-notes" do
    it "extracts notes for a known version" do
      output = `script/release extract-notes v0.13.0`

      expect($?.success?).to be true
      expect(output).to include("Major internal refactor")
    end

    it "fails for unknown version" do
      output = `script/release extract-notes v99.99.99 2>&1`

      expect($?.success?).to be false
      expect(output).to include("No changelog entry found")
    end
  end

  describe "version validation" do
    it "rejects invalid version format" do
      output = `script/release prepare bad-version 2>&1`

      expect($?.success?).to be false
      expect(output).to include("Invalid version")
    end

    it "accepts valid version formats" do
      # Just check validation passes (will fail later due to git state, but that's ok)
      output = `script/release push v99.0.0 2>&1`

      expect(output).not_to include("Invalid version")
    end
  end

  describe "push" do
    it "requires main branch" do
      output = `script/release push v99.0.0 --dry-run 2>&1`

      expect(output).to include("[dry-run]: git branch --show-current")
      expect(output).to include("[dry-run]: git push origin tag v99.0.0")
      expect($?.success?).to be true
    end
  end

  describe "prepare" do
    it "shows what would be prepared in dry-run mode" do
      output = `script/release prepare v99.0.0 --dry-run 2>&1`

      expect($?.success?).to be true
      expect(output).to include("[dry-run]: Would write changelog for v99.0.0")
    end
  end
end
