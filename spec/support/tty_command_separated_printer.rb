require "tty-command"

# A printer that sends stdout to $stdout and stderr to $stderr
# instead of merging them like the quiet printer does
class SeparatedPrinter < TTY::Command::Printers::Abstract
  def initialize(output, options = {})
    super
    # TTY::Command passes options[:err] if provided
    @err = options[:err] || $stderr
  end

  def print_command_start(cmd, *args)
    # Silent - no command decoration
  end

  def print_command_out_data(cmd, *args)
    message = args.join
    @output.write(message)
    @output.flush
  end

  def print_command_err_data(cmd, *args)
    message = args.join
    @err.write(message)
    @err.flush
  end

  def print_command_exit(cmd, status, runtime, *args)
    # Silent - no exit status decoration
  end
end
