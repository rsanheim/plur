require "json"
require "open3"

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
end