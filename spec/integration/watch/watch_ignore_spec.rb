require "spec_helper"

RSpec.describe "plur watch --ignore flag" do
  include PlurWatchHelper

  describe "CLI flag" do
    it "accepts --ignore flag and logs patterns" do
      result = run_plur_watch(timeout: 2)

      # Default patterns should be logged
      expect(result.err).to include("Global watch ignore patterns")
      expect(result.err).to include(".git/**")
      expect(result.err).to include("node_modules/**")
    end

    it "accepts custom --ignore patterns" do
      Dir.chdir(default_ruby_dir) do
        args = "#{plur_binary} --debug watch --ignore='vendor/**' --ignore='tmp/**' --timeout 2"
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!(args)

        expect(result.err).to include("Global watch ignore patterns")
        expect(result.err).to include("vendor/**")
        expect(result.err).to include("tmp/**")
      end
    end

    it "works with plur watch run --ignore" do
      Dir.chdir(default_ruby_dir) do
        args = "#{plur_binary} --debug watch run --ignore='custom/**' --timeout 2"
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!(args)

        expect(result.err).to include("custom/**")
      end
    end
  end

  describe "TOML config" do
    it "uses watch-ignore as global ignore patterns", :skip_if_ci do
      with_temp_watch_project do |project_dir|
        project_dir.join(".plur.toml").write(<<~TOML)
          watch-ignore = ["lib/**"]
          use = "rspec"
        TOML

        lib_file = project_dir.join("lib", "calculator.rb")
        original_content = lib_file.read

        result = run_plur_watch(dir: project_dir, timeout: 3) do
          lib_file.write(original_content + "\n# ignored by watch-ignore")
        end

        expect(result.err).to include("Global watch ignore patterns patterns=[lib/**]")
        expect(result.err).not_to include('path="lib/calculator.rb"')
        expect(result.err).not_to include("Executing job")
        expect(result.success?).to be(true)
      end
    end
  end

  describe "help output" do
    it "shows --ignore in plur watch --help" do
      cmd = TTY::Command.new(uuid: false, printer: :null)
      result = cmd.run!("#{plur_binary} watch --help")

      expect(result.out).to include("--ignore")
      expect(result.out).to include("Patterns to ignore from watch events")
    end

    it "shows --ignore in plur watch run --help" do
      cmd = TTY::Command.new(uuid: false, printer: :null)
      result = cmd.run!("#{plur_binary} watch run --help")

      expect(result.out).to include("--ignore")
      expect(result.out).to include("Patterns to ignore from watch events")
    end
  end
end
