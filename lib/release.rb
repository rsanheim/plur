require "json"

class Release
  def initialize(new_version, prs_in_release = [])
    @new_version = new_version
    @prs_in_release = prs_in_release || find_last_pr_merged_to_main
  end

  def call
    # first, ensure we are on master w/ a clean git status
    # make sure new_version is greater than the current version
    # make sure rux builds cleanly, assuming it does...
    # update changelog based on the 'prs_in_release' info (see update_changelog method below for how to get the info for a single PR)
    # ask the user if they are ready to release with details: old -> new version, changelog edits, prs included...
    # if they confirm, do the release:
    # .  add a git tag matching new_version (so go build will use it)
    # .  rebuild the local binary
    # .  push tags to remote
    # .  create a new release on github
    # .  display success ouptut with URLs for user to confirm
    # if they say no, exit with instructions to help finish the release
    #


  end

  def update_changelog
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

    updated_changelog = changelog.sub(/^# CHANGELOG\n/, "# CHANGELOG\n\n#{new_entry}")
    File.write("CHANGELOG.md", updated_changelog)
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
end
end