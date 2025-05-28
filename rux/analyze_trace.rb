#!/usr/bin/env ruby

require 'json'
require 'optparse'

options = {}
OptionParser.new do |opts|
  opts.banner = "Usage: analyze_trace.rb [trace_file]"
  opts.on("-v", "--verbose", "Show detailed breakdown") { options[:verbose] = true }
end.parse!

trace_file = ARGV[0] || Dir.glob("/tmp/rux-traces/rux-trace-*.json").max_by { |f| File.mtime(f) }

unless trace_file && File.exist?(trace_file)
  puts "No trace file found"
  exit 1
end

puts "Analyzing: #{trace_file}"
puts

events = JSON.parse(File.read(trace_file))

# Group by operation name
by_operation = events.group_by { |e| e["name"] }

# Calculate stats
stats = by_operation.map do |name, events|
  durations = events.map { |e| e["duration_ms"] }
  {
    name: name,
    count: events.size,
    total_ms: durations.sum,
    avg_ms: durations.sum / events.size,
    max_ms: durations.max,
    min_ms: durations.min
  }
end.sort_by { |s| -s[:total_ms] }

# Print summary
puts "Operation Summary:"
puts "-" * 80
puts "%-30s %8s %10s %10s %10s %10s" % ["Operation", "Count", "Total(ms)", "Avg(ms)", "Min(ms)", "Max(ms)"]
puts "-" * 80

stats.each do |stat|
  puts "%-30s %8d %10.2f %10.2f %10.2f %10.2f" % [
    stat[:name],
    stat[:count],
    stat[:total_ms],
    stat[:avg_ms],
    stat[:min_ms],
    stat[:max_ms]
  ]
end

if options[:verbose]
  puts "\nDetailed Breakdown:"
  puts "-" * 80
  
  # Show per-worker stats for run_spec_file
  spec_events = events.select { |e| e["name"] == "run_spec_file" }
  if spec_events.any?
    puts "\nPer-spec execution times:"
    spec_events.sort_by { |e| -e["duration_ms"] }.each do |event|
      puts "  %-50s %10.2f ms (worker %d)" % [
        event["spec_file"],
        event["duration_ms"],
        event["worker_id"] || 0
      ]
    end
  end
  
  # Show process spawn times
  spawn_events = events.select { |e| e["name"] == "process_spawn" }
  if spawn_events.any?
    puts "\nProcess spawn times:"
    spawn_events.each do |event|
      puts "  %-50s %10.2f ms" % [
        event["spec_file"],
        event["duration_ms"]
      ]
    end
  end
end

# Calculate overhead
total_time = events.find { |e| e["name"] == "main.total_execution" }&.fetch("duration_ms", 0)
parallel_time = events.find { |e| e["name"] == "run_specs_parallel" }&.fetch("duration_ms", 0)
spec_time = events.select { |e| e["name"] == "run_spec_file" }.map { |e| e["duration_ms"] }.max || 0

if total_time > 0 && spec_time > 0
  puts "\nTiming Analysis:"
  puts "-" * 80
  puts "Total execution time:     %10.2f ms" % total_time
  puts "Parallel execution time:  %10.2f ms" % parallel_time
  puts "Longest spec file:        %10.2f ms" % spec_time
  puts "Rux overhead:             %10.2f ms (%.1f%%)" % [
    parallel_time - spec_time,
    ((parallel_time - spec_time) / parallel_time * 100)
  ]
end