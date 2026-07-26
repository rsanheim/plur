require "spec_helper"
require "stringio"

# Exercises the Ruby plugin plur embeds into minitest runs. It is loaded
# directly rather than through a minitest run so the IO semantics can be
# compared against a real IO, line by line.
require ROOT_PATH.join("internal", "framework", "minitest", "plur_plugin.rb").to_s

RSpec.describe Minitest::PlurTaggedIO do
  # What a plain IO would have written, one tagged line per newline.
  def reference(&block)
    io = StringIO.new
    io.binmode
    block.call(io)
    io.string.each_line.map { |line| "PLUR_OUT:" + (line.end_with?("\n") ? line : line + "\n") }.join
  end

  def tagged(&block)
    sink = StringIO.new
    sink.binmode
    io = described_class.new(sink)
    block.call(io)
    io.flush_partial
    sink.string
  end

  # IO#puts has fiddly semantics; the plugin has to match them exactly.
  {
    "a string" => ->(io) { io.puts "hello" },
    "a string that already ends in a newline" => ->(io) { io.puts "hello\n" },
    "no arguments" => ->(io) { io.puts },
    "nil" => ->(io) { io.puts nil },
    "an array" => ->(io) { io.puts ["a", "b"] },
    "a nested array" => ->(io) { io.puts ["a", ["b", "c"]] },
    "an empty array" => ->(io) { io.puts [] },
    "several arguments" => ->(io) { io.puts "a", "b" },
    "a non-string" => ->(io) { io.puts 42 },
    "a multiline string" => ->(io) { io.puts "one\ntwo\nthree" }
  }.each do |description, script|
    it "matches IO#puts with #{description}" do
      expect(tagged(&script)).to eq(reference(&script))
    end
  end

  {
    "print with no newline" => ->(io) { io.print "abc" },
    "print then puts on the same line" => ->(io) {
      io.print "ab"
      io.puts "cd"
    },
    "write with several arguments" => ->(io) { io.write "a", "b", "\n" },
    "printf" => ->(io) { io.printf("%05.2f|%s\n", 3.14159, "x") },
    "chained <<" => ->(io) { io << "a" << "b" << "\n" },
    "putc" => ->(io) {
      io.putc 65
      io.putc "\n"
    },
    "utf-8" => ->(io) { io.puts "héllo ☃" },
    "binary bytes" => ->(io) { io.write "\xff\xfe\x00binary\n".b },
    "a carriage return" => ->(io) { io.puts "a\r" },
    "a line longer than any buffer" => ->(io) { io.puts "x" * 300_000 }
  }.each do |description, script|
    it "matches IO semantics for #{description}" do
      expect(tagged(&script)).to eq(reference(&script))
    end
  end

  it "returns what IO returns" do
    io = described_class.new(StringIO.new)

    expect(io.write("abc")).to eq(3)
    expect(io.write("héllo")).to eq("héllo".bytesize)
    expect(io.puts("x")).to be_nil
    expect(io.print("x")).to be_nil
    expect(io.putc("z")).to eq("z")
    expect(io << "x").to equal(io)
  end

  it "holds back a partial line until it is terminated" do
    sink = StringIO.new
    io = described_class.new(sink)

    io.print "PARTIAL"
    expect(sink.string).to eq("")

    io.puts "_DONE"
    expect(sink.string).to eq("PLUR_OUT:PARTIAL_DONE\n")
  end

  it "flushes a trailing partial line once, at the end" do
    sink = StringIO.new
    io = described_class.new(sink)

    io.print "TRAILING"
    io.flush_partial
    io.flush_partial

    expect(sink.string).to eq("PLUR_OUT:TRAILING\n")
  end

  it "keeps lines whole when many threads write at once" do
    sink = StringIO.new
    sink.binmode
    lock = Mutex.new
    # Serialize the sink itself so only the class under test is under test
    guarded = Object.new
    guarded.define_singleton_method(:write) { |str| lock.synchronize { sink.write(str) } }
    io = described_class.new(guarded)

    8.times.map { |i|
      Thread.new { 300.times { |j| io.puts "t#{i}-line#{j}" } }
    }.each(&:join)

    lines = sink.string.lines
    expect(lines.size).to eq(2400)
    expect(lines.grep_v(/\APLUR_OUT:t\d-line\d+\n\z/)).to be_empty
  end
end
