task "db:setup" do
  env_num = ENV['TEST_ENV_NUMBER'] || ''
  STDERR.puts "Error: Database connection failed for test#{env_num}"
  STDERR.puts "Could not connect to server: Connection refused"
  STDERR.puts "Is the server running on host 'localhost' and accepting TCP/IP connections on port 5432?"
  exit 1
end