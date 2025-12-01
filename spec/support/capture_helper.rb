module CaptureHelper
  # Silence stdout/stderr during tests, and return result of the block
  def silence
    stdout = StringIO.new
    stderr = StringIO.new
    begin
      orig_stdout = $stdout
      orig_stderr = $stderr

      $stdout = stdout
      $stderr = stderr
      yield
    ensure
      $stdout = orig_stdout
      $stderr = orig_stderr
    end
  end

  # Silence stdout/stderr during tests, and return whatever was written to them
  def capture(stdout: StringIO.new, stderr: StringIO.new)
    fake_stdout = stdout
    fake_stderr = stderr
    result = begin
      orig_stdout = $stdout
      orig_stderr = $stderr

      $stdout = fake_stdout
      $stderr = fake_stderr
      yield
    ensure
      $stdout = orig_stdout
      $stderr = orig_stderr
    end
    [fake_stdout.string, fake_stderr.string, result]
  end
end
