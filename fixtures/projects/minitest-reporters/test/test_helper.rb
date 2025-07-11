require 'minitest/autorun'
require 'minitest/reporters'

# Use the default reporter (same as example-auth)
Minitest::Reporters.use!(Minitest::Reporters::DefaultReporter.new(color: true))