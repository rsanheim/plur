module Backspin
  class DubplateFormatError < StandardError; end
  class NoMoreRecordingsError < StandardError; end

  class Dubplate
    attr_reader :path, :commands

    def initialize(path)
      @path = path
      @commands = []
      @playback_index = 0
      load_from_file if File.exist?(@path)
    end

    def add_command(command)
      @commands << command
      self
    end

    def save
      FileUtils.mkdir_p(File.dirname(@path))
      dubplate_data = @commands.map(&:to_h)
      File.write(@path, dubplate_data.to_yaml)
    end

    def reload
      @commands = []
      @playback_index = 0
      load_from_file if File.exist?(@path)
      @playback_index = 0  # Reset again after loading to ensure it's at 0
    end

    def exists?
      File.exist?(@path)
    end

    def empty?
      @commands.empty?
    end

    def size
      @commands.size
    end

    def next_command
      if @playback_index >= @commands.size
        raise NoMoreRecordingsError, "No more recordings available for replay"
      end

      command = @commands[@playback_index]
      @playback_index += 1
      command
    end

    def clear
      @commands = []
      @playback_index = 0
    end

    def self.load_or_create(path)
      new(path)
    end

    private

    def load_from_file
      data = YAML.load_file(@path)
      
      unless data.is_a?(Array)
        raise DubplateFormatError, "Invalid dubplate format: expected array but got #{data.class}"
      end

      @commands = data.map { |command_data| Command.from_h(command_data) }
    rescue Psych::SyntaxError => e
      raise DubplateFormatError, "Invalid dubplate format: #{e.message}"
    end
  end
end