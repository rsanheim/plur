require "time"

require_relative "../plur"

class Plur::Changelog
  def initialize(new_version, prs_in_release = [], changelog_input:)
    @new_version = new_version
    @prs_in_release = prs_in_release
    @changelog = changelog_input.read
  end

  def generate_updated_content
    new_entry = generate_entry
    @changelog.sub(/^(## Unreleased\s*\n)/, "\\1#{new_entry}")
  end

  def generate_entry
    date = Time.now.strftime("%Y-%m-%d")
    entry_lines = ["## #{@new_version} - #{date}"]

    if @prs_in_release.empty?
      entry_lines << "* Various improvements and bug fixes"
    else
      @prs_in_release.each do |pr|
        entry_lines << "* #{pr[:title]} [##{pr[:number]}](#{pr[:url]})"
      end
    end

    entry_lines.join("\n") + "\n"
  end
end
