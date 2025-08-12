task "db:setup" do
  env_num = ENV["TEST_ENV_NUMBER"] || ""
  case env_num
  when "1", ""
    STDERR.puts "Error: Database test#{env_num} already exists"
    exit 1
  when "2"
    STDERR.puts "Error: Permission denied for database test2"
    exit 1
  else
    puts "Successfully created test#{env_num}"
  end
end