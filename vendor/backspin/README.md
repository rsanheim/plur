# Backspin

Record and replay CLI interactions for testing, inspired by [VCR](https://github.com/vcr/vcr).

## Overview

Backspin is a Ruby library for characterization testing of command-line interfaces. While VCR records and replays HTTP interactions, Backspin records and replays CLI interactions - capturing stdout, stderr, and exit status from shell commands.

The name "Backspin" evokes the DJ technique of quickly reversing a record to replay a section - perfect for a tool that records and replays command outputs.

## Installation

Add this line to your application's Gemfile:

```ruby
gem 'backspin'
```

And then execute:

    $ bundle install

Or install it yourself as:

    $ gem install backspin

## Usage

### Recording CLI interactions

```ruby
require 'backspin'

# Record a command's output
result = Backspin.record("echo_hello") do
  stdout, stderr, status = Open3.capture3("echo hello")
end

# The cassette is saved to tmp/backspin/echo_hello.yaml
```

### Verifying CLI output

```ruby
# Verify that a command produces the expected output
result = Backspin.verify(cassette: "echo_hello") do
  Open3.capture3("echo hello")
end

expect(result.verified?).to be true
```

### Using verify! for automatic test failures

```ruby
# Automatically fail the test if output doesn't match
Backspin.verify!(cassette: "echo_hello") do
  Open3.capture3("echo hello")
end
```

### Playback mode for fast tests

```ruby
# Return recorded output without running the command
result = Backspin.verify(cassette: "slow_command", mode: :playback) do
  Open3.capture3("slow_command")  # Not actually executed!
end
```

### Custom matchers

```ruby
# Use custom logic to verify output
Backspin.verify(cassette: "version_check", 
                matcher: ->(recorded, actual) {
                  # Just check that both start with "ruby"
                  recorded["stdout"].start_with?("ruby") && 
                  actual["stdout"].start_with?("ruby")
                }) do
  Open3.capture3("ruby --version")
end
```

### VCR-style use_cassette

```ruby
# Record on first run, replay on subsequent runs
Backspin.use_cassette("my_command", record: :once) do
  Open3.capture3("echo hello")
end
```

### Credential Scrubbing

By default, Backspin automatically scrubs common credential patterns from cassettes:

```ruby
# This will automatically scrub AWS keys, API tokens, passwords, etc.
Backspin.record("aws_command") do
  Open3.capture3("aws s3 ls")
end

# Disable credential scrubbing
Backspin.configure do |config|
  config.scrub_credentials = false
end

# Add custom patterns to scrub
Backspin.configure do |config|
  config.add_credential_pattern(/MY_SECRET_[A-Z0-9]+/)
end
```

Automatic scrubbing includes:
- AWS access keys, secret keys, and session tokens
- Google API keys and OAuth client IDs
- Generic API keys, auth tokens, and passwords
- Private keys (RSA, etc.)

## Features

- **Simple recording**: Capture stdout, stderr, and exit status
- **Flexible verification**: Strict matching, playback mode, or custom matchers
- **Auto-naming**: Automatically generate cassette names from RSpec examples
- **Multiple commands**: Record sequences of commands in a single cassette
- **RSpec integration**: Works seamlessly with RSpec's mocking framework
- **Human-readable**: YAML cassettes are easy to read and edit
- **Credential scrubbing**: Automatically removes sensitive data like API keys and passwords

## Development

After checking out the repo, run `bin/setup` to install dependencies. Then, run `rake spec` to run the tests.

To install this gem onto your local machine, run `bundle exec rake install`. To release a new version, update the version number in `version.rb`, and then run `bundle exec rake release`, which will create a git tag for the version, push git commits and the created tag, and push the `.gem` file to [rubygems.org](https://rubygems.org).

## Contributing

Bug reports and pull requests are welcome on GitHub at https://github.com/yourusername/backspin.

## License

The gem is available as open source under the terms of the [MIT License](https://opensource.org/licenses/MIT).