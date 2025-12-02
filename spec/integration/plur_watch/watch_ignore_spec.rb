require "spec_helper"

RSpec.describe "plur watch --ignore flag" do
  include PlurWatchHelper

  describe "CLI flag" do
    it "accepts --ignore flag and logs patterns" do
      result, _, _ = capture_watch_output(plur_timeout: 2)

      # Default patterns should be logged
      expect(result.err).to include("Global watch ignore patterns")
      expect(result.err).to include(".git/**")
      expect(result.err).to include("node_modules/**")
    end

    it "accepts custom --ignore patterns" do
      Dir.chdir(default_ruby_dir) do
        args = "plur --debug watch --ignore='vendor/**' --ignore='tmp/**' --timeout 2"
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!(args)

        expect(result.err).to include("Global watch ignore patterns")
        expect(result.err).to include("vendor/**")
        expect(result.err).to include("tmp/**")
      end
    end

    it "works with plur watch run --ignore" do
      Dir.chdir(default_ruby_dir) do
        args = "plur --debug watch run --ignore='custom/**' --timeout 2"
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!(args)

        expect(result.err).to include("custom/**")
      end
    end
  end

  describe "help output" do
    it "shows --ignore in plur watch --help" do
      cmd = TTY::Command.new(uuid: false, printer: :null)
      result = cmd.run!("plur watch --help")

      expect(result.out).to include("--ignore")
      expect(result.out).to include("Patterns to ignore from watch events")
    end

    it "shows --ignore in plur watch run --help" do
      cmd = TTY::Command.new(uuid: false, printer: :null)
      result = cmd.run!("plur watch run --help")

      expect(result.out).to include("--ignore")
      expect(result.out).to include("Patterns to ignore from watch events")
    end
  end
end
