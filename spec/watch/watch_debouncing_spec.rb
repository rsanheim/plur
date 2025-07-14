require "spec_helper"

RSpec.describe "plur watch debouncing" do
  # Debouncing tests are now consolidated into watch_spec.rb
  # This file can be removed as its functionality is covered elsewhere

  # The following tests have been moved:
  # - "uses default debounce of 100ms when not specified" -> watch_spec.rb
  # - "respects custom debounce delay" -> watch_spec.rb
  # - "handles multiple file changes with debouncing" -> watch_integration_spec.rb
end
