require "json"
require "open3"
require_relative "changelog"

module Plur
  module Release
    module Helpers
      def run_or_dry_run(cmd)
        if dry_run
          puts "[dry-run]: #{cmd}"
        else
          run(cmd)
        end
      end

      def system_or_dry_run(cmd, exception: true)
        if dry_run
          puts "[dry-run]: #{cmd}"
        else
          system(cmd, exception: exception)
        end
      end

      def run(cmd)
        stdout, stderr, status = Open3.capture3(cmd)
        abort "Error running '#{cmd}': #{stderr}" unless status.success?
        stdout.strip
      end

      def last_release_tag
        run("gh release list --limit 1 --json tagName -q '.[0].tagName'").tap do |tag|
          return nil if tag.empty?
        end
      end

      def find_prs_since_last_release(dry_run: false)
        if dry_run
          return [{number: 123, title: "Example PR", url: "https://github.com/example/pr/123"}]
        end

        tag = last_release_tag
        commit_range = tag ? "#{tag}..main" : "main"

        prs = []
        run("git log #{commit_range} --merges --oneline").each_line do |line|
          if line =~ /Merge pull request #(\d+)/
            prs << fetch_pr_info($1)
          end
        end
        prs.compact
      end

      def fetch_pr_info(pr_number)
        data = JSON.parse(run("gh pr view #{pr_number} --json number,title,url"))
        {number: data["number"], title: data["title"], url: data["url"]}
      end
    end

    class Prepare
      include Helpers

      def initialize(version, dry_run: false)
        @version = version
        @dry_run = dry_run
      end

      def call
        prs = find_prs_since_last_release(dry_run: @dry_run)
        changelog = Plur::Changelog.new(@version, prs)

        if @dry_run
          puts "[dry-run]: Would write changelog for #{@version} with #{prs.size} PRs"
          puts "[dry-run]: PRs: #{prs.map { |pr| "##{pr[:number]}" }.join(", ")}" unless prs.empty?
        else
          File.write("CHANGELOG.md", changelog.generate_updated_content)
          print_summary(prs)
        end
      end

      private

      def print_summary(prs)
        pr_list = prs.empty? ? "  (no PRs found)" : prs.map { |pr| "  ##{pr[:number]} - #{pr[:title]}" }.join("\n")

        puts <<~MSG

          ============================================================
          RELEASE PREPARED: #{@version}
          ============================================================

          PRs included:
          #{pr_list}

          CHANGELOG.md updated. Next steps:
            1. Review/edit CHANGELOG.md if needed
            2. Run: script/release push #{@version}
          ============================================================
        MSG
      end
    end

    class Push
      include Helpers

      attr_reader :version, :dry_run

      def initialize(version, dry_run: false)
        @version = version
        @dry_run = dry_run
      end

      def call
        verify_on_main!
        verify_clean!
        verify_changelog!
        tag_and_push!

        print_success
      end

      private

      def verify_on_main!
        branch = run_or_dry_run("git branch --show-current")
        abort "Error: Must be on 'main' branch to release (currently on '#{branch}')" unless dry_run || branch == "main"
        puts "✓ On main branch"
      end

      def verify_clean!
        status = run_or_dry_run("git status --porcelain")
        abort "Error: Uncommitted changes. Commit or stash first.\n#{status}" unless dry_run || status.empty?
        puts "✓ Git status clean"
      end

      def verify_changelog!
        unless dry_run
          abort "Error: CHANGELOG.md not found" unless File.exist?("CHANGELOG.md")
          content = File.read("CHANGELOG.md")
          abort "Error: No changelog entry for #{@version}. Run 'script/release prepare #{@version}' first." unless content.include?("## #{@version}")
        end
        puts "✓ Changelog entry found for #{@version}"
      end

      def tag_and_push!
        puts "→ Creating tag #{@version}..."
        system_or_dry_run("git tag -a #{@version} -m 'Release #{@version}'", exception: true)

        puts "→ Pushing tag..."
        system_or_dry_run("git push origin tag #{@version}", exception: true)
      end

      def print_success
        puts <<~MSG

          ============================================================
          ✓ RELEASE TAGGED: #{@version}
          ============================================================

          GitHub Actions will build the release.
          Watch: https://github.com/rsanheim/plur/actions
          ============================================================
        MSG
      end
    end
  end
end
