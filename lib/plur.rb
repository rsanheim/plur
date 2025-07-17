# frozen_string_literal: true

# Plur - Project-Level Unified Resources
# Shared constants and configuration for builds, integration specs, etc

require "pathname"
require "singleton"

module Plur
  def self.config
    Config.instance
  end
end

require_relative "plur/config"