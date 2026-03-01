module Plur
  class Config
    include Singleton

    attr_reader :edant_watcher_version, :root_dir, :watcher_dir, :fixtures_dir, :default_ruby_dir, :default_rails_dir

    def initialize
      # we should automate getting the latest version from https://github.com/e-dant/watcher
      @edant_watcher_version = "0.13.8"

      @root_dir = Pathname.new(__dir__).join("../..").expand_path
      @watcher_dir = @root_dir.join("embedded", "watcher")

      @fixtures_dir = @root_dir.join("fixtures", "projects")
      @default_ruby_dir = @fixtures_dir.join("default-ruby")
      @default_rails_dir = @fixtures_dir.join("default-rails")
    end

    # lock things down a bit for CI
    def plur_cores
      ENV["CI"] ? 4 : 8
    end
  end
end
