# frozen_string_literal: true

require "tty-command"
require "fileutils"
require "pathname"
require_relative "../plur"

module Plur
  class Install
    ARCHITECTURES = {
      "x86_64" => "amd64",
      "aarch64" => "arm64",
      "arm64" => "arm64"
    }.freeze

    def initialize
      @cmd = TTY::Command.new(printer: :null)
    end

    def run(command:, service:, compose_prefix: nil, binary: nil)
      container_name = build_container_name(service, compose_prefix)

      unless container_exists?(container_name)
        puts "Error: Could not find running container: #{container_name}"
        puts "Make sure the container is running"
        exit 1
      end

      case command
      when "status"
        show_status(container_name)
      else
        install(container_name, binary)
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

    def find_or_build_binary(arch)
      dist_dir = Plur.config.root_dir.join("dist")
      binary_path = dist_dir.join("plur-linux-#{arch}")

      unless dist_dir.exist? && binary_path.exist?
        puts ">>> Building Linux binaries..."
        require "rake"
        Rake.application.init
        Rake.application.load_rakefile
        Rake::Task["build:linux"].invoke
      end

      unless binary_path.exist?
        puts "Error: Binary not found at #{binary_path}"
        puts "Run 'bin/rake build:linux' to build the binaries first."
        exit 1
      end

      binary_path
    end

    def install(container_name, binary_path)
      if binary_path.nil?
        puts ">>> Detecting container architecture..."
        arch = detect_architecture(container_name)
        puts "    Architecture: linux/#{arch}"
        binary_path = find_or_build_binary(arch)
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

      # Check existing version
      existing_version = get_plur_version(container_name)
      if existing_version && !existing_version.include?("not found")
        puts "    Replacing existing plur: #{existing_version}"
      end

      # Create temp directory
      puts ">>> Creating temp directory in container..."
      result = @cmd.run("docker exec #{container_name} mktemp -d")
      temp_dir = result.out.strip
      puts "    Temp directory: #{temp_dir}"

      # Copy binary
      puts ">>> Copying binary to container..."
      @cmd.run("docker cp #{binary_path} #{container_name}:#{temp_dir}/plur")

      # Install binary
      puts ">>> Installing binary in container..."
      install_script = <<~BASH
        set -e
        mkdir -p /usr/local/bin
        mv -f #{temp_dir}/plur /usr/local/bin/plur
        chmod +x /usr/local/bin/plur
        rmdir #{temp_dir}
      BASH
      @cmd.run("docker exec #{container_name} bash -c #{Shellwords.escape(install_script)}")

      # Verify installation
      puts ">>> Verifying installation..."
      version = get_plur_version(container_name)
      if version.nil? || version.include?("not found")
        puts "✗ Installation failed!"
        puts "  Error: #{version}"
        exit 1
      else
        puts "✓ Installation successful!"
        puts "  Version: #{version}"
      end

      # Run doctor
      puts
      puts ">>> Running plur doctor..."
      begin
        doctor_cmd = TTY::Command.new(printer: :quiet)
        doctor_cmd.run("docker exec #{container_name} plur doctor")
      rescue TTY::Command::ExitError
        # Doctor might fail but installation was successful
      end

      puts
      puts "✓ plur has been successfully installed on #{container_name}"
      puts "  You can now use: docker exec #{container_name} plur [arguments]"
    end

    def show_status(container_name)
      puts ">>> Checking plur installation status on container: #{container_name}"
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
        result = @cmd.run("docker exec #{container_name} bash -c 'which plur && ls -lh $(which plur)'", only_output_on_error: true)
        puts result.out
      rescue TTY::Command::ExitError
        puts "  Unable to get binary info"
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
  end
end
