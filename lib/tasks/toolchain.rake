namespace :toolchain do
  desc "Validate toolchain version consistency (.mise.toml, plur/go.mod)"
  task :check do
    sh "mise doctor"
    sh "mise current"

    mise_go = File.read(".mise.toml")[/^\s*go\s*=\s*"(\d+\.\d+)/, 1]
    gomod_go = File.read("plur/go.mod")[/^go\s+(\d+\.\d+)/, 1]

    if mise_go.nil?
      abort("[toolchain:check] Could not find `go = \"x.y.z\"` in .mise.toml")
    end
    if gomod_go.nil?
      abort("[toolchain:check] Could not find `go x.y.z` directive in plur/go.mod")
    end

    if mise_go != gomod_go
      abort("[toolchain:check] Go version mismatch: .mise.toml=#{mise_go}, plur/go.mod=#{gomod_go}")
    end

    if File.exist?(".ruby-version")
      abort("[toolchain:check] .ruby-version should not exist; Ruby version is sourced from .mise.toml")
    end

    puts "[toolchain:check] Toolchain configuration looks consistent"
  end
end
