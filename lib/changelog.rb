class Changelog
  def initialize(new_version, prs_in_release = [])
    @new_version = new_version
    @prs_in_release = prs_in_release || find_last_pr_merged_to_main
  end

  def changelog_updates
    # Get PR information using gh CLI
    pr_info = get_current_pr_info
    # Require a PR to exist
    raise "Error: No PR found for the current branch. Please create a PR before releasing." unless pr_info

    # Update CHANGELOG.md
    changelog = File.read("CHANGELOG.md")
    date = Time.now.strftime("%Y-%m-%d")

    new_entry = <<~ENTRY
      ## #{new_version} - #{date}
      * #{pr_info[:title]} [##{pr_info[:number]}](#{pr_info[:url]})
    ENTRY

    changelog.sub(/^# CHANGELOG\n/, "# CHANGELOG\n\n#{new_entry}")
  end

  # Helper method to get current PR information
  def get_current_pr_info

    # Check if gh command is available
    unless system("which gh > /dev/null 2>&1")
      abort "Error: GitHub CLI (gh) is not installed. Please install it from https://cli.github.com/"
    end

    # Get the current PR information
    pr_json, status = Open3.capture2("gh pr view --json number,title,url")

    return unless status.success? && !pr_json.empty?

    begin
      pr_data = JSON.parse(pr_json)
      {
        number: pr_data["number"],
        title: pr_data["title"],
        url: pr_data["url"]
      }
    rescue JSON::ParserError => e
      abort "Error: Could not parse PR information: #{e.message}"
    end
  end