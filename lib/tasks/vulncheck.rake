namespace :vuln do
  desc "Run govulncheck to scan for known vulnerabilities (strict: all modules)"
  task :check do
    puts "[vuln:check] Running govulncheck (scan=module)..."
    sh "govulncheck", "-scan=module"
  end
end
