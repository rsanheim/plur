# frozen_string_literal: true

require "tty-command"
require "json"
require_relative "../plur"

module Plur
  class DockerInstall
    ARCHITECTURES = {
      "x86_64" => "amd64",
      "aarch64" => "arm64",
      "arm64" => "arm64"
    }.freeze

    def initialize
      @cmd = TTY::Command.new(printer: :null)
    end

    def run(command:, service:, compose_prefix: nil, binary: nil, install_path: nil)
      container_name = build_container_name(service, compose_prefix)

      unless container_exists?(container_name)
        puts "Error: Could not find running container: #{container_name}"
        puts "Make sure the container is running"
        exit 1
      end

      case command
      when "status"
        show_status(container_name, install_path)
      else
        install(container_name, binary, install_path)
      end
    end

    private

    def build_container_name(service, compose_prefix)
      compose_prefix ? "#{compose_prefix}_#{service}_1" : service
    end

    def container_exists?(container_name)
      result = @cmd.run("docker ps --filter name=#{container_name} --format '{{.Names}}'", only_output_on_error: true)
      !result.out.strip.empty?
    rescue TTY::Command::ExitError
      false
    end

    def detect_architecture(container_name)
      result = @cmd.run("docker exec #{container_name} uname -m", only_output_on_error: true)
      arch = result.out.strip

      ARCHITECTURES[arch] || raise("Unknown architecture: #{arch}")
    rescue TTY::Command::ExitError => e
      raise "Failed to detect architecture: #{e.message}"
    end

    def find_binary(arch)
      artifacts_file = Plur.config.root_dir.join("dist", "artifacts.json")

      unless artifacts_file.exist?
        puts "Error: No binaries found. Run 'bin/rake build:all' first."
        exit 1
      end

      artifacts = JSON.parse(artifacts_file.read)

      binary = artifacts.find do |a|
        a["type"] == "Binary" && a["goos"] == "linux" && a["goarch"] == arch
      end

      if binary.nil?
        puts "Error: No linux/#{arch} binary found in dist/"
        puts "Run 'bin/rake build:all' to build binaries for all platforms."
        exit 1
      end

      Plur.config.root_dir.join(binary["path"])
    end

    def install(container_name, binary_path, install_path = nil)
      install_path ||= "/usr/local/bin"

      if binary_path.nil?
        puts ">>> Detecting container architecture..."
        arch = detect_architecture(container_name)
        puts "    Architecture: linux/#{arch}"
        binary_path = find_binary(arch)
      else
        binary_path = Pathname.new(binary_path)
        unless binary_path.exist?
          puts "Error: Binary not found at #{binary_path}"
          exit 1
        end
      end

      binary_size = (binary_path.size / 1024.0 / 1024.0).round(1)
      puts ">>> Installing plur to container: #{container_name}"
      puts "    Binary: #{binary_path} (#{binary_size}M)"
      puts "    Install path: #{install_path}"

      # Check if install path exists
      check_result = @cmd.run("docker exec #{container_name} test -d #{install_path} && echo 'exists' || echo 'missing'", only_output_on_error: true)
      if check_result.out.strip == "missing"
        puts ">>> Creating install directory: #{install_path}"
        @cmd.run("docker exec #{container_name} mkdir -p #{install_path}")
      end

      # Check existing version at install path
      plur_path = "#{install_path}/plur"
      existing_version = get_plur_version_at_path(container_name, plur_path)
      if existing_version && !existing_version.include?("not found")
        puts "    Replacing existing plur: #{existing_version}"
      end

      # Copy binary directly to install path
      puts ">>> Copying binary to container..."
      @cmd.run("docker cp #{binary_path} #{container_name}:#{plur_path}")
      @cmd.run("docker exec #{container_name} chmod +x #{plur_path}")

      # Verify installation
      puts ">>> Verifying installation..."
      version = get_plur_version_at_path(container_name, plur_path)
      if version.nil? || version.include?("not found")
        puts "✗ Installation failed!"
        puts "  Error: #{version}"
        exit 1
      else
        puts "✓ Installation successful!"
        puts "  Version: #{version}"
        puts "  Location: #{plur_path}"
      end

      # Try to install watchr binary using plur watch install
      puts ">>> Installing watchr binary..."
      begin
        @cmd.run("docker exec #{container_name} #{plur_path} watch install")
        puts "✓ Watchr binary installed successfully"
      rescue TTY::Command::ExitError => e
        puts "⚠ Could not install watchr binary: #{e.message.lines.first}"
        puts "  The watch command may not work properly"
      end

      # Run doctor
      puts
      puts ">>> Running plur doctor..."
      begin
        doctor_cmd = TTY::Command.new(printer: :quiet)
        doctor_cmd.run("docker exec #{container_name} #{plur_path} doctor")
      rescue TTY::Command::ExitError
        # Doctor might fail but installation was successful
      end

      puts
      puts "✓ plur has been successfully installed on #{container_name}"
      puts "  You can now use: docker exec #{container_name} #{plur_path} [arguments]"
    end

    def show_status(container_name, install_path = nil)
      install_path ||= "/usr/local/bin"
      plur_path = "#{install_path}/plur"

      puts ">>> Checking plur installation status on container: #{container_name}"
      puts "    Install path: #{install_path}"
      puts

      version = get_plur_version(container_name)
      if version.nil? || version.include?("not found")
        puts "✗ plur is not installed on #{container_name}"
        exit 1
      else
        puts "✓ plur is installed on #{container_name}"
        puts "  Version: #{version}"
      end

      # Binary info
      puts
      puts ">>> Binary information:"
      begin
        # Check primary install location
        result = @cmd.run("docker exec #{container_name} bash -c 'test -f #{plur_path} && ls -lh #{plur_path} || echo \"Not found at #{plur_path}\"'", only_output_on_error: true)
        puts "  Primary: #{result.out.strip}"

        # Check which plur resolves to
        result = @cmd.run("docker exec #{container_name} bash -c 'which plur'", only_output_on_error: true)
        puts "  In PATH: #{result.out.strip}"
      rescue TTY::Command::ExitError
        puts "  Unable to get binary info"
      end

      # Check watchr binary (plur watch install puts it in ~/.cache/plur/bin)
      puts
      puts ">>> Watchr binary:"
      begin
        result = @cmd.run("docker exec #{container_name} bash -c 'which watchr || ls -lh ~/.cache/plur/bin/watchr 2>/dev/null || echo \"Not found\"'", only_output_on_error: true)
        puts "  #{result.out.strip}"
      rescue TTY::Command::ExitError
        puts "  Unable to check watchr binary"
      end

      # Doctor output
      puts
      puts ">>> Running plur doctor..."
      begin
        doctor_cmd = TTY::Command.new(printer: :quiet)
        doctor_cmd.run("docker exec #{container_name} plur doctor")
      rescue TTY::Command::ExitError
        puts "  Error running plur doctor"
      end
    end

    def get_plur_version(container_name)
      result = @cmd.run("docker exec #{container_name} plur --version", only_output_on_error: true)
      result.out.strip
    rescue TTY::Command::ExitError
      nil
    end

    def get_plur_version_at_path(container_name, path)
      result = @cmd.run("docker exec #{container_name} #{path} --version", only_output_on_error: true)
      result.out.strip
    rescue TTY::Command::ExitError
      nil
    end
  end
end
