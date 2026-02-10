require "time"

require_relative "../plur"

class Plur::Changelog
  attr_reader :new_prs

  def initialize(new_version, prs_in_release = [], changelog_input:)
    @new_version = new_version
    @prs_in_release = prs_in_release
    @changelog = changelog_input.read
    @new_prs = []
  end

  def generate_updated_content
    new_entry = generate_entry
    @changelog.sub(/^(## Unreleased\s*\n)/, "\\1#{new_entry}")
  end

  def generate_entry
    date = Time.now.strftime("%Y-%m-%d")
    entry_lines = ["## #{@new_version} - #{date}"]

    existing = existing_pr_numbers
    @new_prs = @prs_in_release.reject { |pr| existing.include?(pr[:number]) }

    if @new_prs.empty?
      entry_lines << "* Various improvements and bug fixes"
    else
      @new_prs.each do |pr|
        entry_lines << "* #{pr[:title]} [##{pr[:number]}](#{pr[:url]})"
      end
    end

    entry_lines.join("\n") + "\n\n"
  end

  def existing_pr_numbers
    @changelog.scan(/\[#(\d+)\]/).flatten.map(&:to_i).to_set
  end
end
