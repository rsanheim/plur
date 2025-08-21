require "spec_helper"

RSpec.describe "Flexible argument ordering" do
  context "with --no-color flag" do
    it "works with --no-color before file arguments" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("--no-color", "--dry-run", "spec/calculator_spec.rb")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 1 spec")
        expect(result.err).to include("--no-color")
      end
    end

    it "works with --no-color after file arguments" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/calculator_spec.rb", "--no-color", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 1 spec")
        expect(result.err).to include("--no-color")
      end
    end
  end

  context "with -n/--workers flag" do
    it "works with -n before file arguments" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("-n", "4", "--dry-run", "spec/calculator_spec.rb")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 1 spec")
      end
    end

    it "works with -n after file arguments" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/calculator_spec.rb", "-n", "4", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 1 spec")
      end
    end

    it "works with --workers after file arguments" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/calculator_spec.rb", "--workers", "4", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 1 spec")
      end
    end
  end

  context "with directory arguments" do
    let(:expected_spec_files) { 12 }

    it "expands spec/ directory before flags" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running #{expected_spec_files} specs")
      end
    end

    it "expands spec/ directory after flags" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("--dry-run", "spec/")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running #{expected_spec_files} specs")
      end
    end

    it "expands spec/ directory with --no-color after" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/", "--no-color", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running #{expected_spec_files} specs")
        expect(result.err).to include("--no-color")
      end
    end
  end

  context "with multiple flags and arguments" do
    it "handles complex argument ordering with flags after files" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("spec/calculator_spec.rb", "spec/counter_spec.rb", "--no-color", "-n", "2", "--dry-run")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 2 specs")
        expect(result.err).to include("--no-color")
        expect(result.err).to include("groups") # Should show grouping with 2 workers
      end
    end

    it "handles complex argument ordering with flags before files" do
      Dir.chdir(project_fixture("default-ruby")) do
        result = run_plur("--no-color", "-n", "2", "--dry-run", "spec/calculator_spec.rb", "spec/counter_spec.rb")

        expect(result.err).to include("plur version")
        expect(result.err).to include("[dry-run] Running 2 specs")
        expect(result.err).to include("--no-color")
        expect(result.err).to include("groups") # Should show grouping with 2 workers
      end
    end
  end
end
