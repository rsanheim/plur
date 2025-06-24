# frozen_string_literal: true

# Plur - Project-Level Unified Resources
# Shared constants and configuration for the Rux build system

require "pathname"
require "singleton"

module Plur
  class Config
    include Singleton

    attr_reader :edant_watcher_version, :root_dir, :rux_dir, :watcher_dir, :local_rux_binary,
      :fixtures_dir, :default_ruby_dir, :default_rails_dir

    def initialize
      # Version information
      @edant_watcher_version = "0.13.6"

      # Directory paths
      @root_dir = Pathname.new(__dir__).expand_path
      @rux_dir = @root_dir.join("rux")
      @watcher_dir = @rux_dir.join("embedded", "watcher")

      # Fixture paths
      @fixtures_dir = @root_dir.join("fixtures", "projects")
      @default_ruby_dir = @fixtures_dir.join("default-ruby")
      @default_rails_dir = @fixtures_dir.join("default-rails")

      # Binary paths
      @local_rux_binary = @rux_dir.join("rux")
    end

    # Runtime configuration
    def rux_cores
      ENV["CI"] ? 3 : 8
    end
  end

  def self.config
    Config.instance
  end
end
