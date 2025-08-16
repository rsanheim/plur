require "spec_helper"

RSpec.describe "plur watch find command" do
  let(:watch_find_fixture) { project_fixture("watch-find-test") }
  let(:dx_project_path) { Pathname.new(__dir__).parent.parent.parent.join("references", "example-project") }

  context "with file that has a working default mapping" do
    it "shows the mapping exists and spec exists" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/standard_mapper.rb")

      expect(result).to be_success
      expect(result.out).to include("✓")
      expect(result.out).to include("lib/standard_mapper.rb")
      expect(result.out).to include("spec/standard_mapper_spec.rb")
      expect(result.out).to include("exists via:")
    end

    it "works with Rails app structure" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "app/services/payment_processor.rb")

      expect(result).to be_success
      expect(result.out).to include("✓")
      expect(result.out).to include("app/services/payment_processor.rb")
      expect(result.out).to include("spec/services/payment_processor_spec.rb")
    end
  end

  context "with lib file in project with misaligned spec structure" do
    it "detects the mismatch and suggests alternative specs" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/misaligned_mapper.rb")

      expect(result).to be_success
      expect(result.out).to include("✗")
      expect(result.out).to include("mapping exists but spec not found")
      expect(result.out).to include("Found alternative specs:")
      expect(result.out).to include("spec/lib/misaligned_mapper_spec.rb")
    end

    it "suggests a corrected mapping rule for misaligned structure" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/misaligned_mapper.rb")

      expect(result).to be_success
      expect(result.out).to include("Suggested rule based on found specs:")
      expect(result.out).to include('pattern = "lib/**/*.rb"')
      # For files directly in lib/, the pattern doesn't include {path}
      expect(result.out).to include('target = "spec/lib/{name}_spec.rb"')
    end

    it "handles nested misaligned structures" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/utils/formatter.rb")

      expect(result).to be_success
      expect(result.out).to include("✗")
      expect(result.out).to include("lib/utils/formatter.rb → spec/utils/formatter_spec.rb")
      expect(result.out).to include("mapping exists but spec not found")
      expect(result.out).to include("Found alternative specs:")
      expect(result.out).to include("spec/lib/utils/formatter_spec.rb")
    end
  end

  context "with file that has no default mapping" do
    it "searches for potential specs and suggests locations" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "config/routes.rb")

      expect(result).to be_success
      expect(result.out).to include("✗ No mapping for: config/routes.rb")
      
      # Should suggest locations for new specs since none exist
      expect(result.out).to include("No existing specs found")
    end
  end

  context "with file where no spec exists" do
    it "indicates no spec exists and suggests where to create one" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/no_spec_file.rb")

      expect(result).to be_success
      expect(result.out).to include("✗")
      expect(result.out).to include("lib/no_spec_file.rb → spec/no_spec_file_spec.rb")
      expect(result.out).to include("mapping exists but spec not found")
      # Since no alternatives exist, it should not show any
    end
  end

  context "with --dry-run flag" do
    it "shows what rules would be added without modifying config" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", "lib/misaligned_mapper.rb", "--dry-run")

      expect(result).to be_success
      expect(result.out).to include("Found alternative specs:")
      expect(result.out).to include("spec/lib/misaligned_mapper_spec.rb")
      expect(result.out).to include("Suggested rule based on found specs:")
      
      # The current implementation shows the rule but doesn't have special dry-run output
      # when the mapping fails and alternatives are found
      expect(result.out).to include('pattern = "lib/**/*.rb"')
      expect(result.out).to include('target = "spec/lib/{name}_spec.rb"')
    end
  end

  context "with interactive mode" do
    it "prompts for confirmation when adding rules" do
      # Use Open3 directly for interactive testing
      cmd = "echo 'n' | plur -C #{watch_find_fixture} watch find lib/misaligned_mapper.rb -i"
      stdout, stderr, status = Open3.capture3(cmd)

      expect(status).to be_success
      expect(stdout).to include("Add 1 mapping rule(s) to .plur.toml? [y/N]:")
      expect(stdout).to include("No changes made")
    end
  end

  context "with multiple files" do
    it "processes each file and reports findings" do
      result = run_plur("-C", watch_find_fixture, "watch", "find", 
                       "lib/standard_mapper.rb", "lib/misaligned_mapper.rb")

      expect(result).to be_success
      expect(result.out).to include("lib/standard_mapper.rb")
      expect(result.out).to include("lib/misaligned_mapper.rb")
      expect(result.out).to include("✓") # standard_mapper works
      expect(result.out).to include("✗") # misaligned_mapper has issues
    end
  end

  context "with real-world example-project project" do
    it "detects misaligned lib structure in example-project project" do
      result = run_plur("-C", dx_project_path, "watch", "find", "lib/example-project/cli.rb")

      expect(result).to be_success
      expect(result.out).to include("✗")
      expect(result.out).to include("mapping exists but spec not found")
      expect(result.out).to include("Found alternative specs:")
      expect(result.out).to include("spec/lib/example-project/cli_spec.rb")
      expect(result.out).to include("Suggested rule based on found specs:")
      expect(result.out).to include('pattern = "lib/**/*.rb"')
      expect(result.out).to include('target = "spec/lib/{path}/{name}_spec.rb"')
    end
  end
end