module Backspin
  module RSpecMetadata
    def self.dubplate_name_from_example
      return nil unless defined?(RSpec) && RSpec.respond_to?(:current_example)

      example = RSpec.current_example
      return nil unless example

      # Build path from example group hierarchy
      path_parts = []

      # Traverse up through parent example groups
      current_group = example.example_group
      while current_group&.respond_to?(:metadata) && current_group.metadata
        description = current_group.metadata[:description]
        path_parts.unshift(sanitize_filename(description)) if description && !description.empty?
        current_group = current_group.superclass if current_group.superclass.respond_to?(:metadata)
      end

      # Add the example description
      path_parts << sanitize_filename(example.description)

      path_parts.join("/")
    end

    class << self
      private

      def sanitize_filename(str)
        # Replace spaces with underscores and remove special characters
        str.gsub(/\s+/, "_").gsub(/[^a-zA-Z0-9_-]/, "")
      end
    end
  end
end
