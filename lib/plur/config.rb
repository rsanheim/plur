module Plur
  class Config
    include Singleton

    attr_reader :edant_watcher_version, :root_dir, :plur_dir, :watcher_dir,
      :fixtures_dir, :default_ruby_dir, :default_rails_dir

    def initialize
      # Version information
      @edant_watcher_version = "0.13.8"

      # Directory paths
      @root_dir = Pathname.new(__dir__).join("../..").expand_path
      @plur_dir = @root_dir.join("plur")
      @watcher_dir = @plur_dir.join("embedded", "watcher")

      # Fixture paths
      @fixtures_dir = @root_dir.join("fixtures", "projects")
      @default_ruby_dir = @fixtures_dir.join("default-ruby")
      @default_rails_dir = @fixtures_dir.join("default-rails")
    end

    # Runtime configuration
    def plur_cores
      ENV["CI"] ? 4 : 8
    end
  end
end
