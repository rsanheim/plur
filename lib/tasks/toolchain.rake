require "open3"

namespace :toolchain do
  desc "Validate toolchain version consistency (.mise.toml, go.mod)"
  task :check do
    if system("which mise > /dev/null 2>&1")
      output, status = Open3.capture2e("mise", "doctor")
      unless status.success?
        warn output
        abort("[toolchain:check] mise doctor reported problems (exit #{status.exitstatus})")
      end
    end

    mise_go = File.read(".mise.toml")[/^\s*go\s*=\s*"(\d+\.\d+)/, 1]
    gomod_go = File.read("go.mod")[/^go\s+(\d+\.\d+)/, 1]

    if mise_go.nil?
      abort("[toolchain:check] Could not find `go = \"x.y.z\"` in .mise.toml")
    end
    if gomod_go.nil?
      abort("[toolchain:check] Could not find `go x.y.z` directive in go.mod")
    end

    if mise_go != gomod_go
      abort("[toolchain:check] Go version mismatch: .mise.toml=#{mise_go}, go.mod=#{gomod_go}")
    end
  end
end
