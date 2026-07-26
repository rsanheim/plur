# frozen_string_literal: true

require "stringio"

module Minitest
  # Tags everything the tests themselves write to stdout with a marker so plur
  # can tell it apart from minitest's progress characters, which minitest writes
  # without trailing newlines. Progress stays bare: it is the common case and
  # should cost nothing.
  #
  # Minitest's reporters capture the real $stdout when they are constructed,
  # which happens before Minitest.plugin_plur_init runs, so replacing $stdout
  # here never touches minitest's own output.
  class PlurTaggedIO
    PREFIX = "PLUR_OUT:"

    def initialize(io)
      @io = io
      @pending = +"".b
      @tagging = true
      @mutex = Mutex.new
    end

    # Formatting is delegated to StringIO so puts/print/write/printf keep Ruby's
    # exact IO semantics (flattened arrays, nil, no-arg puts, and the "don't
    # double the newline" rule) instead of a hand-rolled approximation.
    def write(*args)
      scratch { |io| io.write(*args) }
    end

    def print(*args)
      scratch { |io| io.print(*args) }
    end

    def puts(*args)
      scratch { |io| io.puts(*args) }
    end

    def printf(*args)
      scratch { |io| io.printf(*args) }
    end

    def putc(char)
      scratch { |io| io.putc(char) }
    end

    def <<(arg)
      scratch { |io| io << arg }
      self
    end

    def flush
      @io.flush
      self
    end

    def to_io
      @io
    end

    # capture_subprocess_io saves stdout with dup and restores it with reopen.
    # Duping the wrapper itself would hand out a second handle on the real
    # stdout, and closing it would take stdout down with it.
    def dup
      self.class.new(@io.dup)
    end

    # While stdout is redirected somewhere else (a Tempfile, for capture) the
    # bytes are not going to plur, so they must not be tagged. Reopening from a
    # dup of this wrapper is the restore side of that dance.
    def reopen(target, *rest)
      flush_partial
      @mutex.synchronize { @tagging = target.is_a?(self.class) }
      @io.reopen(target, *rest)
      self
    end

    # Emits the trailing partial line (a `print` with no newline). Called at the
    # end of the run; safe to call more than once.
    def flush_partial
      @mutex.synchronize do
        next if @pending.empty?
        @io.write(tagged(@pending + "\n".b))
        @pending.clear
      end
    end

    def respond_to_missing?(name, include_all = false)
      @io.respond_to?(name, include_all)
    end

    def method_missing(name, *args, **kwargs, &blk)
      @io.send(name, *args, **kwargs, &blk)
    end

    private

    def scratch
      buffer = StringIO.new
      buffer.binmode
      result = yield buffer
      emit(buffer.string)
      result
    end

    # Buffers until a newline arrives, then writes whole tagged lines.
    def emit(chunk)
      @mutex.synchronize do
        next @io.write(chunk) unless @tagging
        @pending << chunk
        cut = @pending.rindex("\n".b)
        next if cut.nil?
        @io.write(tagged(@pending.slice!(0, cut + 1)))
      end
    end

    # Prefixes every line in chunk; chunk always ends with a newline.
    def tagged(chunk)
      out = +"".b
      chunk.each_line { |line| out << PREFIX << line }
      out
    end
  end

  def self.plugin_plur_init(_options)
    return if $stdout.is_a?(PlurTaggedIO) # a second Minitest.run would double-tag

    tagged = PlurTaggedIO.new($stdout)
    $stdout = tagged
    # STDOUT.puts would otherwise sidestep the wrapper and land in plur's output
    # untagged, stranded on the progress line.
    Object.send(:remove_const, :STDOUT)
    Object.const_set(:STDOUT, tagged)
    Minitest.after_run { tagged.flush_partial }
  end
end
