require "spec_helper"

# The output mode contract. `auto` resolves to summary over a pipe (how agents
# and CI run plur) and to progress under a terminal; an explicit mode wins
# regardless of where stdout goes:
#
#   --output=auto|progress|summary  >  PLUR_OUTPUT  >  config `output = "..."`  >  auto
#
# Summary mode drops only the per-example markers (`.`, `F`, `*`, `E`) and the
# newline that ends them. Test output, failure details, rerun commands, the
# framework summary, and the exit status are untouched. Color is a separate
# axis and resolves on its own.
RSpec.describe "Output mode" do
  # Lines made only of progress markers, color stripped. A test that prints a
  # bare dot on its own line would count too, so the fixtures here print words.
  def marker_lines(out)
    out.gsub(ansi, "").lines.map(&:chomp).grep(/\A[.F*E]+\z/)
  end

  def run_failing(*args, env: {})
    chdir(project_fixture("failing_specs")) do
      run_plur_allowing_errors(*args, "-n", "2", "spec/mixed_results_spec.rb", env: env)
    end
  end

  def run_passing(*args, env: {})
    chdir(default_ruby_dir) { run_plur(*args, "-n", "2", "spec/calculator_spec.rb", env: env) }
  end

  context "auto over a pipe" do
    it "emits no markers on a passing run and keeps the summary" do
      result = run_passing

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("5 examples, 0 failures")
      expect(result.exit_status).to eq(0)
    end

    it "keeps rspec pending, failure details, rerun commands, and the exit status" do
      result = run_failing

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("Pending:")
      expect(result.out).to include("# Not implemented yet")
      expect(result.out).to include("Failures:")
      expect(result.out).to include("1) MixedResults some pass, some fail fails this test")
      expect(result.out).to include("Failure/Error: expect(\"foo\").to eq(\"bar\")")
      expect(result.out).to include("8 examples, 2 failures, 3 pending")
      expect(result.out).to include("Failed examples:")
      expect(result.out).to include("rspec ./spec/mixed_results_spec.rb:7 # MixedResults some pass, some fail fails this test")
      expect(result.out).to include("rspec ./spec/mixed_results_spec.rb:15 # MixedResults some pass, some fail has an error")
      expect(result.exit_status).to eq(1)
    end

    it "keeps minitest failure details and the exit status" do
      result = chdir(project_fixture!("minitest-failures")) do
        Bundler.with_unbundled_env { run_plur_allowing_errors("--use", "minitest", "-n", "2") }
      end

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("Failures:")
      expect(result.out).to include("MixedResultsTest#test_display_name_failure [test/mixed_results_test.rb:46]")
      expect(result.out).to include("ArrayOperationsTest#test_find_max_with_nil:")
      expect(result.out).to include("ArgumentError: comparison of Integer with nil failed")
      expect(result.out).to include("13 runs, 16 assertions, 6 failures, 1 error, 0 skips")
      expect(result.exit_status).to eq(1)
    end

    it "still streams what the suite writes to stdout" do
      Dir.mktmpdir do |tmpdir|
        File.write(File.join(tmpdir, "puts_spec.rb"), <<~SPEC)
          RSpec.describe "puts under summary mode" do
            it "prints" do
              puts "HELLO_FROM_PUTS"
              expect(true).to be true
            end
          end
        SPEC

        result = chdir(tmpdir) { run_plur("puts_spec.rb") }

        expect(result.out).to include("HELLO_FROM_PUTS")
        expect(marker_lines(result.out)).to be_empty
        expect(result.out).to include("1 example, 0 failures")
        expect(result.exit_status).to eq(0)
      end
    end
  end

  context "explicit progress over a pipe" do
    it "emits plain markers when color is auto" do
      result = run_failing("--output=progress")

      expect(marker_lines(result.out)).to eq([".F.F.***"])
      expect(result.out).not_to match(ansi)
      expect(result.out).to include("8 examples, 2 failures, 3 pending")
      expect(result.exit_status).to eq(1)
    end

    it "colors markers only when color is forced" do
      result = run_failing("--output=progress", "--color=always")

      expect(result.out).to include("\e[32m.\e[0m")
      expect(result.out).to include("\e[31mF\e[0m")
      expect(result.out).to include("\e[33m*\e[0m")
    end

    it "parses after the positional target like --color" do
      result = chdir(default_ruby_dir) { run_plur("spec/calculator_spec.rb", "--output=progress") }

      expect(marker_lines(result.out)).to eq(["....."])
    end
  end

  context "explicit summary over a pipe" do
    it "suppresses markers and keeps the results" do
      result = run_failing("--output=summary")

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("8 examples, 2 failures, 3 pending")
      expect(result.exit_status).to eq(1)
    end

    it "with forced color, colors the remaining output" do
      result = run_failing("--output=summary", "--color=always")

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("\e[31mFailure/Error: expect(\"foo\").to eq(\"bar\")\e[0m")
    end
  end

  context "configuration" do
    def with_output_config(value)
      Dir.mktmpdir do |dir|
        config_path = File.join(dir, "output.toml")
        File.write(config_path, %(output = "#{value}"\n))
        yield config_path
      end
    end

    it "PLUR_OUTPUT=progress emits markers over a pipe" do
      result = run_passing(env: {"PLUR_OUTPUT" => "progress"})

      expect(marker_lines(result.out)).to eq(["....."])
    end

    it "the --output flag beats PLUR_OUTPUT" do
      result = run_passing("--output=summary", env: {"PLUR_OUTPUT" => "progress"})

      expect(marker_lines(result.out)).to be_empty
    end

    it "config file output = \"progress\" emits markers over a pipe" do
      with_output_config("progress") do |config_path|
        result = run_passing(env: {"PLUR_CONFIG_FILE" => config_path})

        expect(marker_lines(result.out)).to eq(["....."])
      end
    end

    it "env beats config: PLUR_OUTPUT=summary beats output = \"progress\"" do
      with_output_config("progress") do |config_path|
        result = run_passing(env: {"PLUR_CONFIG_FILE" => config_path, "PLUR_OUTPUT" => "summary"})

        expect(marker_lines(result.out)).to be_empty
      end
    end

    it "rejects an unknown mode" do
      result = chdir(default_ruby_dir) { run_plur_allowing_errors("--output=bogus") }

      expect(result.exit_status).not_to eq(0)
      expect(result.err).to include("progress")
      expect(result.err).to include("summary")
    end
  end

  context "under a terminal", :pty do
    def run_failing_in_pty(*args)
      run_in_pty(plur_binary, *args, "-n", "2", "spec/mixed_results_spec.rb", chdir: project_fixture("failing_specs"))
    end

    it "auto keeps the progress markers" do
      result = run_failing_in_pty

      expect(marker_lines(result.out)).to eq([".F.F.***"])
      expect(result.out).to include("\e[31mF\e[0m")
      expect(result.exit_status).to eq(1)
    end

    it "explicit progress keeps the markers" do
      result = run_failing_in_pty("--output=progress")

      expect(marker_lines(result.out)).to eq([".F.F.***"])
    end

    it "explicit summary suppresses markers and keeps the results" do
      result = run_failing_in_pty("--output=summary")

      expect(marker_lines(result.out)).to be_empty
      expect(result.out).to include("Failures:")
      expect(result.out).to include("8 examples, 2 failures, 3 pending")
      expect(result.out).to include("rspec ./spec/mixed_results_spec.rb:7")
      expect(result.exit_status).to eq(1)
    end
  end

  context "in a real terminal", :tmux do
    it "auto shows the progress markers on screen" do
      tmux_terminal(dir: default_ruby_dir) do |terminal|
        terminal.submit("#{plur_binary} -n 2 spec/calculator_spec.rb")
        expect(terminal.wait_for("5 examples, 0 failures", timeout: 30)).to be(true), "run never finished:\n#{terminal.screen}"

        expect(marker_lines(terminal.screen)).to eq(["....."])
      end
    end
  end
end
