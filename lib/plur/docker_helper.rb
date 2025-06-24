# frozen_string_literal: true

require_relative "../../plur"

# Helper module for Docker operations using Plur configuration
module DockerHelper
  extend self

  def docker_run(command, platform: nil, interactive: false, volumes: [], env: {})
    args = ["docker", "run"]

    # Basic flags
    args << "-it" if interactive
    args << "--rm"

    # Volumes from Plur
    args << "-v #{Plur.config.root_dir}:/workspace:rw"
    args << "-v /workspace/references"
    args << "-v /workspace/vendor"
    volumes.each { |v| args << "-v #{v}" }

    # Working directory
    args << "-w /workspace"

    # Environment from Plur
    args << "-e RUX_CORES=#{Plur.config.rux_cores}"
    args << "-e EDANT_WATCHER_VERSION=#{Plur.config.edant_watcher_version}"
    args << "-e TERM=xterm-256color"
    env.each { |k, v| args << "-e #{k}=#{v}" }

    # Image and command
    args << image_name(platform: platform)
    args << command

    args.join(" ")
  end

  def docker_compose_run(service, command = nil, env: {})
    # Set Plur environment variables
    ENV["RUX_CORES"] = Plur.config.rux_cores.to_s
    ENV["EDANT_WATCHER_VERSION"] = Plur.config.edant_watcher_version
    env.each { |k, v| ENV[k] = v.to_s }

    args = ["docker-compose", "run", "--rm"]
    args << service
    args << command if command

    args.join(" ")
  end

  def image_name(platform: nil)
    if platform && !["latest", nil].include?(platform)
      "rux-test:#{platform}"
    else
      "rux-test:latest"
    end
  end

  def image_exists?(platform: nil)
    name = image_name(platform: platform)
    system("docker images | grep -q '#{name.split(":").first}.*#{name.split(":").last}'",
      out: File::NULL, err: File::NULL)
  end

  def build_image(platform: nil)
    # Always use buildx for consistency
    platform_arg = platform ? "--platform #{platform}" : ""
    cmd = "docker buildx build #{platform_arg} -t #{image_name} .".strip

    system(cmd)
  end
end
