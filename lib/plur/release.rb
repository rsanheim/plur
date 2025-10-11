require "json"
require "open3"
require_relative "changelog"

class Plur::Release
  def initialize(new_version, options = {})
    @new_version = new_version
    @prs_in_release = options[:prs_in_release] || find_last_pr_merged_to_main
    @automated = options[:automated] || false
    @draft = options.fetch(:draft, true) # default to draft for safety
    @main_required = false
  end

  def call
    # Ensure we are on main branch with clean git status
    ensure_on_main_branch! if @main_required
    ensure_clean_git_status!

    # Get current version and ensure new version is greater
    current_version = get_current_version
    ensure_version_is_newer!(current_version, @new_version)

    # Build plur to ensure it compiles cleanly
    ensure_plur_builds!

    # Update changelog
    changelog = Plur::Changelog.new(@new_version, @prs_in_release)
    updated_content = changelog.generate_updated_content
    File.write("CHANGELOG.md", updated_content)

    # Show release summary and ask for confirmation
    show_release_summary(current_version, @new_version, @prs_in_release)

    unless @automated || confirm_release?
      puts "\nRelease cancelled. To continue later:"
      puts "  1. Review and update CHANGELOG.md if needed"
      puts "  2. Run: script/release #{@new_version}"
      exit 0
    end

    if @automated
      puts "\n>>> Running in automated mode - skipping confirmation"
    end

    # Perform the release
    perform_release!

    # Show success message
    show_success_message
  end

  private

  def capture3!(cmd, err: nil)
    stdout, stderr, status = Open3.capture3(cmd)
    unless status.success?
      warn err if err
      warn "Error: #{stderr}"
      abort
    end
    [stdout, stderr, status]
  end

  def ensure_on_main_branch!
    stdout, _, _ = capture3!("git branch --show-current")
    current_branch = stdout.strip
    abort "Error: You must be on the 'main' branch to release. Currently on '#{current_branch}'" unless current_branch == "main"
  end

  def ensure_clean_git_status!
    stdout, _, _ = capture3!("git status --porcelain")
    status = stdout.strip

    unless status.empty?
      warn "Error: Git working directory is not clean."
      warn "Uncommitted changes:"
      warn status
      warn "\nPlease commit or stash your changes."
      abort
    end
  end

  def get_current_version
    _, = capture3!("git describe --tags --abbrev=0")[0].strip
  end

  def ensure_version_is_newer!(current, new)
    require "rubygems/version"
    current_v = Gem::Version.new(current.sub(/^v/, ""))
    new_v = Gem::Version.new(new.sub(/^v/, ""))

    abort "Error: New version #{new} must be greater than current version #{current}" unless new_v > current_v
  end

  def ensure_plur_builds!
    puts "Building plur to ensure it compiles..."
    Dir.chdir("plur") do
      system("go build -mod=mod -o plur .", exception: true)
    end
    puts "✓ plur builds successfully"
  end

  def show_release_summary(current_version, new_version, prs)
    pr_list = if prs.empty?
      "  (no PRs found since last release)"
    else
      prs.map { |pr| "  ##{pr[:number]} - #{pr[:title]}" }.join("\n")
    end

    puts <<~SUMMARY

      #{"=" * 60}
      RELEASE SUMMARY
      #{"=" * 60}
      Current version: #{current_version}
      New version:     #{new_version}

      PRs included in this release:
      #{pr_list}

      Changelog has been updated. Please review CHANGELOG.md
      #{"=" * 60}
    SUMMARY
  end

  def confirm_release?
    print "\nProceed with release? (y/N): "
    response = $stdin.gets.chomp.downcase
    response == "y" || response == "yes"
  end

  def perform_release!
    puts "\nPerforming release..."

    # Commit changelog if there are changes
    if system("git diff --quiet CHANGELOG.md")
      puts "  ✓ No changelog changes to commit"
    else
      puts "  → Committing changelog..."
      system("git add CHANGELOG.md", exception: true)
      system("git commit -m 'Update CHANGELOG for #{@new_version}'", exception: true)
    end

    # Extract release notes for GoReleaser
    puts "  → Extracting release notes..."
    extract_release_notes_to_file!(@new_version)

    # Use GoReleaser to handle everything:
    # - Create and push tag
    # - Build multi-platform binaries
    # - Generate checksums
    # - Create archives
    # - Create GitHub release
    # - Upload artifacts
    puts "  → Running GoReleaser..."

    # Build GoReleaser command with draft flag
    goreleaser_cmd = "goreleaser release --release-notes=../.goreleaser-notes.md --clean"
    unless @draft
      goreleaser_cmd += " --draft=false"
      puts "    (Publishing as non-draft release)"
    else
      puts "    (Creating draft release)"
    end

    Dir.chdir("plur") do
      system(goreleaser_cmd, exception: true)
    end

    # Push commits if any (GoReleaser already pushed the tag)
    if system("git diff --quiet origin/main..HEAD")
      puts "  ✓ No commits to push"
    else
      puts "  → Pushing commits to main..."
      system("git push origin main", exception: true)
    end
  end

  def extract_release_notes_to_file!(version)
    notes = extract_changelog_entry(version)
    File.write(".goreleaser-notes.md", notes)
  end

  def extract_changelog_entry(version)
    content = File.read("CHANGELOG.md")
    # Extract the section for this version
    version_escaped = Regexp.escape(version)
    if content =~ /^## #{version_escaped}.*?\n(.*?)(?=^## |\z)/m
      $1.strip
    else
      "Release #{version}"
    end
  end

  def show_success_message
    puts <<~SUCCESS

      #{"=" * 60}
      🎉 RELEASE SUCCESSFUL!
      #{"=" * 60}

      Version #{@new_version} has been released!

      Next steps:
        • View release: https://github.com/rsanheim/plur/releases/tag/#{@new_version}
        • Test installation: go install github.com/rsanheim/plur/plur@#{@new_version}
        • Announce the release
      #{"=" * 60}
    SUCCESS
  end

  def find_last_pr_merged_to_main
    last_tag = get_current_version
    commit_range = (last_tag == "v0.0.0") ? "main" : "#{last_tag}..main"

    stdout, _, _ = capture3!("git log #{commit_range} --merges --oneline")
    prs = []
    stdout.each_line do |commit|
      if commit =~ /Merge pull request #(\d+)/
        pr_info = get_pr_info($1)
        prs << pr_info if pr_info
      end
    end

    prs
  end

  def get_pr_info(pr_number)
    stdout, = capture3!("gh pr view #{pr_number} --json number,title,url")

    data = JSON.parse(stdout)
    {
      number: data["number"],
      title: data["title"],
      url: data["url"]
    }
  end
end
