require "spec_helper"

RSpec.describe "--exclude-pattern" do
  # Build a self-contained fixture project so tests do not depend on the
  # shape of any existing fixtures/projects/*. Each test gets a fresh
  # tmpdir with the spec layout we want to exercise.
  def setup_rspec_fixture(dir)
    FileUtils.mkdir_p(dir)
    %w[
      spec/system/login_spec.rb
      spec/system/checkout_spec.rb
      spec/legacy/old_spec.rb
      spec/models/user_spec.rb
      spec/models/post_spec.rb
    ].each do |relative|
      path = File.join(dir, relative)
      FileUtils.mkdir_p(File.dirname(path))
      File.write(path, "# stub spec, never executed in dry-run\n")
    end
  end

  def setup_minitest_fixture(dir)
    FileUtils.mkdir_p(dir)
    %w[
      test/integration/api_test.rb
      test/integration/web_test.rb
      test/models/user_test.rb
      test/models/post_test.rb
    ].each do |relative|
      path = File.join(dir, relative)
      FileUtils.mkdir_p(File.dirname(path))
      File.write(path, "# stub test, never executed in dry-run\n")
    end
  end

  def write_config(dir, contents)
    File.write(File.join(dir, ".plur.toml"), contents)
  end

  describe "CLI flag" do
    it "excludes matching files for rspec under autodiscovery" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=rspec",
            "--exclude-pattern", "spec/system/**/*_spec.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("matched no selected files")
        expect(result.err).not_to include("spec/system/login_spec.rb")
        expect(result.err).not_to include("spec/system/checkout_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
        expect(result.err).to include("spec/models/post_spec.rb")
        expect(result.err).to include("spec/legacy/old_spec.rb")
      end
    end

    it "ignores CLI exclude patterns that match no selected files" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=rspec",
            "--exclude-pattern", "*user*/_spec.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("[warn]")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end

    it "excludes matching files for minitest" do
      Dir.mktmpdir do |dir|
        setup_minitest_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=minitest",
            "--exclude-pattern", "test/integration/**/*_test.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("test/integration/api_test.rb")
        expect(result.err).not_to include("test/integration/web_test.rb")
        expect(result.err).to include("test/models/user_test.rb")
        expect(result.err).to include("test/models/post_test.rb")
      end
    end

    it "ORs multiple --exclude-pattern flags together" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=rspec",
            "--exclude-pattern", "spec/system/**/*_spec.rb",
            "--exclude-pattern", "spec/legacy/**/*_spec.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("spec/system/")
        expect(result.err).not_to include("spec/legacy/")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end

    it "errors when excludes filter every candidate" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--use=rspec",
            "--exclude-pattern", "spec/**/*_spec.rb"
          )
        end

        expect(result).not_to be_success
        expect(result.err).to include("no test files remain after applying exclude patterns")
      end
    end

    it "applies excludes to explicit directory inputs" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=rspec",
            "spec/models",
            "--exclude-pattern", "spec/models/post_spec.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("spec/models/post_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end

    it "excludes explicit focused file:line targets by the underlying spec file" do
      chdir(project_fixture("rspec-success-simple")) do
        result = run_plur_allowing_errors(
          "--dry-run",
          "--use=rspec",
          "spec/simple_spec.rb:2",
          "--exclude-pattern", "spec/simple_spec.rb"
        )

        expect(result).not_to be_success
        expect(result.err).to include("no test files remain after applying exclude patterns")
        expect(result.err).not_to include("spec/simple_spec.rb:2")
      end
    end
  end

  describe "TOML config" do
    it "honors exclude_patterns from .plur.toml" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        write_config(dir, <<~TOML)
          [job.rspec]
          exclude_patterns = ["spec/system/**/*_spec.rb"]
        TOML

        result = Dir.chdir(dir) do
          run_plur_allowing_errors("--dry-run", "--use=rspec")
        end

        expect(result).to be_success
        expect(result.err).not_to include("spec/system/")
        expect(result.err).to include("spec/legacy/old_spec.rb")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end

    it "does not warn when a config exclude pattern matches no selected files" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        write_config(dir, <<~TOML)
          [job.rspec]
          exclude_patterns = ["spec/nope/**/*_spec.rb"]
        TOML

        result = Dir.chdir(dir) do
          run_plur_allowing_errors("--dry-run", "--use=rspec")
        end

        expect(result).to be_success
        expect(result.err).not_to include("matched no selected files")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end

    it "appends CLI excludes on top of config excludes (additive, not override)" do
      Dir.mktmpdir do |dir|
        setup_rspec_fixture(dir)
        write_config(dir, <<~TOML)
          [job.rspec]
          exclude_patterns = ["spec/system/**/*_spec.rb"]
        TOML

        result = Dir.chdir(dir) do
          run_plur_allowing_errors(
            "--dry-run",
            "--use=rspec",
            "--exclude-pattern", "spec/legacy/**/*_spec.rb"
          )
        end

        expect(result).to be_success
        expect(result.err).not_to include("spec/system/")
        expect(result.err).not_to include("spec/legacy/")
        expect(result.err).to include("spec/models/user_spec.rb")
      end
    end
  end
end
