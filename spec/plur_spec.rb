# frozen_string_literal: true

require "spec_helper"
require_relative "../plur"

RSpec.describe Plur::Config do
  let(:config) { Plur::Config.instance }

  describe "singleton behavior" do
    it "returns the same instance" do
      expect(config).to be(Plur::Config.instance)
      expect(config).to be(Plur.config)
    end
  end

  describe "constants" do
    it "defines edant_watcher_version as a non-empty string" do
      expect(config.edant_watcher_version).to be_a(String)
      expect(config.edant_watcher_version).not_to be_empty
      expect(config.edant_watcher_version).to match(/^\d+\.\d+\.\d+$/)
    end

    it "defines root_dir as an existing Pathname" do
      expect(config.root_dir).to be_a(Pathname)
      expect(config.root_dir).to exist
      expect(config.root_dir).to be_directory
    end

    it "defines plur_dir as an existing Pathname" do
      expect(config.plur_dir).to be_a(Pathname)
      expect(config.plur_dir).to exist
      expect(config.plur_dir).to be_directory
      expect(config.plur_dir.basename.to_s).to eq("plur")
    end

    it "defines watcher_dir as a Pathname (may not exist yet)" do
      expect(config.watcher_dir).to be_a(Pathname)
      expect(config.watcher_dir.to_s).to include("embedded/watcher")
    end

    it "defines local_plur_binary as a Pathname" do
      expect(config.local_plur_binary).to be_a(Pathname)
      expect(config.local_plur_binary.basename.to_s).to eq("plur")
      expect(config.local_plur_binary.dirname).to eq(config.plur_dir)
    end
  end

  describe "path relationships" do
    it "has plur_dir as a child of root_dir" do
      expect(config.plur_dir.to_s).to start_with(config.root_dir.to_s)
    end

    it "has watcher_dir as a child of plur_dir" do
      expect(config.watcher_dir.to_s).to start_with(config.plur_dir.to_s)
    end

    it "has local_plur_binary inside plur_dir" do
      expect(config.local_plur_binary.to_s).to start_with(config.plur_dir.to_s)
    end
  end
end
