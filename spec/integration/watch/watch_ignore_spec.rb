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

  describe "watch find parity" do
    it "applies custom ignore patterns to watch find" do
      Dir.chdir(default_ruby_dir) do
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!("#{plur_binary} watch find --ignore='spec/**' spec/calculator_spec.rb")

        expect(result.exit_status).to eq(2)
        expect(result.out).to include("msg=ignored")
        expect(result.out).not_to include('msg="found rules"')
      end
    end

    it "applies default ignore patterns to watch find" do
      Dir.chdir(default_ruby_dir) do
        cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
        result = cmd.run!("#{plur_binary} watch find node_modules/calculator_spec.rb")

        expect(result.exit_status).to eq(2)
        expect(result.out).to include("msg=ignored")
      end
    end

    it "rejects files outside the project instead of planning them" do
      Dir.mktmpdir do |outside_dir|
        outside_file = File.join(outside_dir, "foo.go")
        File.write(outside_file, "package foo\n")

        Dir.chdir(default_ruby_dir) do
          cmd = TTY::Command.new(timeout: 5, uuid: false, printer: :null)
          result = cmd.run!("#{plur_binary} -u go-test watch find #{outside_file}")

          expect(result.exit_status).to eq(2)
          expect(result.out).to include("msg=ignored")
          expect(result.out).not_to include('msg="would run"')
        end
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
