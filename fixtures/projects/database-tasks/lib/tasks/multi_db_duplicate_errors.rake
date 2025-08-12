task "db:setup" do
  # All workers will fail with the same error
  STDERR.puts "Error: PostgreSQL is not installed"
  STDERR.puts "Please install PostgreSQL and try again"
  exit 1
end