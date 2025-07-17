require "time"

require_relative "../plur"

class Plur::Changelog
  def initialize(new_version, prs_in_release = [])
    @new_version = new_version
    @prs_in_release = prs_in_release
  end

  def generate_updated_content
    content = File.read("CHANGELOG.md")
    new_entry = generate_entry
    content.sub(/^(# .*CHANGELOG\s*\n)/, "\\1\n#{new_entry}\n")
  end

  def generate_entry
    date = Time.now.strftime("%Y-%m-%d")
    entry_lines = ["## #{@new_version} - #{date}"]

    if @prs_in_release.empty?
      entry_lines << "\n* Various improvements and bug fixes"
    else
      entry_lines << ""
      @prs_in_release.each do |pr|
        entry_lines << "* #{pr[:title]} [##{pr[:number]}](#{pr[:url]})"
      end
    end

    entry_lines.join("\n") + "\n"
  end
end
