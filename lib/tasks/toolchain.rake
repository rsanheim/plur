require "open3"
require "yaml"

namespace :toolchain do
  desc "Validate toolchain version consistency across development, CI, and fixtures"
  task :check do
    if system("which mise > /dev/null 2>&1")
      output, status = Open3.capture2e("mise", "doctor")
      unless status.success?
        warn output
        abort("[toolchain:check] mise doctor reported problems (exit #{status.exitstatus})")
      end
    end

    mise_go = File.read(".mise.toml")[/^\s*go\s*=\s*"(\d+\.\d+\.\d+)"/, 1]
    gomod_go = File.read("go.mod")[/^go\s+(\d+\.\d+\.\d+)$/, 1]

    if mise_go.nil?
      abort("[toolchain:check] Could not find `go = \"x.y.z\"` in .mise.toml")
    end
    if gomod_go.nil?
      abort("[toolchain:check] Could not find `go x.y.z` directive in go.mod")
    end

    if mise_go != gomod_go
      abort("[toolchain:check] Go version mismatch: .mise.toml=#{mise_go}, go.mod=#{gomod_go}")
    end

    mise_ruby = File.read(".mise.toml")[/^ruby\s*=\s*"(\d+\.\d+\.\d+)"/, 1]
    abort("[toolchain:check] Could not find Ruby version in .mise.toml") unless mise_ruby

    circle = YAML.load_file(".circleci/config.yml")
    versions = {
      "CircleCI Go" => [circle.dig("parameters", "go_version", "default"), mise_go],
      "CircleCI Ruby" => [circle.dig("parameters", "ruby_version", "default"), mise_ruby],
      "Rails .ruby-version" => [File.read("fixtures/projects/default-rails/.ruby-version").strip, mise_ruby]
    }
    release = File.read(".github/workflows/release.yml")
    versions["Release Go"] = [release[/go-version: "([^"]+)"/, 1], mise_go]
    versions["Release Ruby"] = [release[/ruby-version: "([^"]+)"/, 1], mise_ruby]
    %w[Dockerfile .devcontainer/Dockerfile].each do |file|
      version = File.read("fixtures/projects/default-rails/#{file}")[/^ARG RUBY_VERSION=(.+)$/, 1]
      versions["Rails #{file}"] = [version, mise_ruby]
    end
    versions.each do |name, (actual, expected)|
      abort("[toolchain:check] #{name} version mismatch: #{actual.inspect}, .mise.toml=#{expected}") unless actual == expected
    end
  end
end
