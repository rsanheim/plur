require "spec_helper"

RSpec.describe "Exclude pattern CLI" do
  it "excludes files during autodiscovery" do
    chdir(default_ruby_dir) do
      result = run_plur("--dry-run", "--exclude-pattern", "spec/models/**/*_spec.rb")

      expect(result).to be_success
      expect(result.err).not_to include("spec/models/user_spec.rb")
      expect(result.err).not_to include("spec/models/system_spec.rb")
      expect(result.err).to include("spec/calculator_spec.rb")
    end
  end

  it "applies repeated exclude patterns" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--dry-run",
        "--exclude-pattern", "spec/models/**/*_spec.rb",
        "--exclude-pattern", "spec/services/**/*_spec.rb"
      )

      expect(result).to be_success
      expect(result.err).not_to include("spec/models/user_spec.rb")
      expect(result.err).not_to include("spec/services/email_service_spec.rb")
      expect(result.err).to include("spec/calculator_spec.rb")
    end
  end

  it "errors when excludes remove every selected file" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors(
        "--dry-run",
        "spec/models",
        "--exclude-pattern", "spec/models/**/*_spec.rb"
      )

      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("no test files remain after applying exclude patterns")
    end
  end

  it "excludes files with explicit directory input" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--dry-run",
        "spec/models",
        "--exclude-pattern", "spec/models/system_spec.rb"
      )

      expect(result).to be_success
      expect(result.err).to include("spec/models/user_spec.rb")
      expect(result.err).not_to include("spec/models/system_spec.rb")
    end
  end

  it "shows discovery and exclusion info through the logger in dry-run mode" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--debug",
        "--dry-run",
        "--exclude-pattern", "spec/models/**/*_spec.rb"
      )

      expect(result).to be_success
      expect(result.err).to include("[dry-run]")
      expect(result.err).to include("test file discovery")
      expect(result.err).to include("excluded=2")
      expect(result.err).to include("remaining=")
    end
  end

  it "shows discovery event in real debug runs without dry-run prefix" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--debug",
        "spec/models",
        "--exclude-pattern", "spec/models/system_spec.rb"
      )

      expect(result).to be_success
      expect(result.err).to include("test file discovery")
      expect(result.err).not_to include("[dry-run] test file discovery")
    end
  end

  it "excludes files when using an explicit include glob" do
    chdir(default_ruby_dir) do
      result = run_plur(
        "--dry-run",
        "spec/models/*_spec.rb",
        "--exclude-pattern", "spec/models/system_spec.rb"
      )

      expect(result).to be_success
      expect(result.err).to include("spec/models/user_spec.rb")
      expect(result.err).not_to include("spec/models/system_spec.rb")
    end
  end

  it "errors on nonexistent file before applying excludes" do
    chdir(default_ruby_dir) do
      result = run_plur_allowing_errors(
        "--dry-run",
        "spec/does_not_exist_spec.rb",
        "--exclude-pattern", "spec/**/*_spec.rb"
      )

      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("file not found")
      expect(result.err).not_to include("no test files remain after applying exclude patterns")
    end
  end

  it "supports --exclude alias" do
    chdir(default_ruby_dir) do
      result = run_plur("--dry-run", "--exclude", "spec/models/**/*_spec.rb")

      expect(result).to be_success
      expect(result.err).not_to include("spec/models/user_spec.rb")
      expect(result.err).to include("spec/calculator_spec.rb")
    end
  end

  it "works with minitest framework" do
    chdir(project_fixture("minitest-success")) do
      result = run_plur("--use", "minitest", "--dry-run", "--exclude-pattern", "test/string_helper_test.rb")

      expect(result).to be_success
      expect(result.err).to include("test/calculator_test.rb")
      expect(result.err).not_to include("test/string_helper_test.rb")
    end
  end
end
