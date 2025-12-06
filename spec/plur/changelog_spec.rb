require "spec_helper"
require "stringio"
require "plur/changelog"

RSpec.describe Plur::Changelog do
  let(:existing_changelog) do
    <<~CHANGELOG
      # plur CHANGELOG

      ## Unreleased

      ## v0.17.0 - 2025-12-01
      * Previous feature [#100](https://github.com/rsanheim/plur/pull/100)
    CHANGELOG
  end

  before do
    allow(Time).to receive(:now).and_return(Time.new(2025, 12, 6))
  end

  describe "#generate_updated_content" do
    it "inserts new entry after Unreleased section" do
      prs = [{number: 123, title: "New feature", url: "https://github.com/rsanheim/plur/pull/123"}]
      changelog = described_class.new("v0.18.0", prs, changelog_input: StringIO.new(existing_changelog))

      result = changelog.generate_updated_content

      expect(result).to match(/## Unreleased\n\n## v0\.18\.0/)
    end

    it "keeps Unreleased at the top" do
      prs = [{number: 123, title: "New feature", url: "https://github.com/rsanheim/plur/pull/123"}]
      changelog = described_class.new("v0.18.0", prs, changelog_input: StringIO.new(existing_changelog))

      result = changelog.generate_updated_content

      lines = result.lines
      unreleased_line = lines.index { |l| l.include?("## Unreleased") }
      new_version_line = lines.index { |l| l.include?("## v0.18.0") }

      expect(unreleased_line).to be < new_version_line
    end

    it "does not add extra blank line after version header" do
      prs = [{number: 123, title: "New feature", url: "https://github.com/rsanheim/plur/pull/123"}]
      changelog = described_class.new("v0.18.0", prs, changelog_input: StringIO.new(existing_changelog))

      result = changelog.generate_updated_content

      expect(result).to include("## v0.18.0 - 2025-12-06\n* New feature")
    end
  end

  describe "#generate_entry" do
    it "formats PRs correctly" do
      prs = [{number: 123, title: "New feature", url: "https://github.com/rsanheim/plur/pull/123"}]
      changelog = described_class.new("v0.18.0", prs, changelog_input: StringIO.new(""))

      entry = changelog.generate_entry

      expect(entry).to eq("## v0.18.0 - 2025-12-06\n* New feature [#123](https://github.com/rsanheim/plur/pull/123)\n")
    end

    it "handles multiple PRs" do
      prs = [
        {number: 123, title: "First PR", url: "https://github.com/rsanheim/plur/pull/123"},
        {number: 124, title: "Second PR", url: "https://github.com/rsanheim/plur/pull/124"}
      ]
      changelog = described_class.new("v0.18.0", prs, changelog_input: StringIO.new(""))

      entry = changelog.generate_entry

      expect(entry).to include("* First PR [#123]")
      expect(entry).to include("* Second PR [#124]")
    end

    it "handles empty PR list" do
      changelog = described_class.new("v0.18.0", [], changelog_input: StringIO.new(""))

      entry = changelog.generate_entry

      expect(entry).to include("Various improvements and bug fixes")
    end
  end
end
