require "spec_helper"

RSpec.describe "Backspin record naming", pending: "Auto-naming removed from record method - keeping for future reference" do
  let(:record_dir) { Pathname.new("tmp/backspin_data") }

  before do
    # Use tmp for integration tests
    Backspin.configure do |config|
      config.backspin_dir = record_dir
    end
    FileUtils.rm_rf(record_dir)
  end

  describe "auto-generated names" do
    it "handles simple example names" do
      result = Backspin.call do
        Open3.capture3("echo test")
      end

      expect(result.record_path.to_s).to match(%r{Backspin_record_naming/auto-generated_names/handles_simple_example_names\.yaml$})
    end

    it "handles names with special characters" do
      result = Backspin.call do
        Open3.capture3("echo 'special chars!'")
      end

      expect(result.record_path.to_s).to match(%r{handles_names_with_special_characters\.yaml$})
    end

    context "nested contexts" do
      context "deeply nested" do
        it "includes all context levels" do
          result = Backspin.call do
            Open3.capture3("echo nested")
          end

          expect(result.record_path.to_s).to match(%r{Backspin_record_naming/auto-generated_names/nested_contexts/deeply_nested/includes_all_context_levels\.yaml$})
        end
      end
    end
  end

  after do
    # Clean up records created during tests
    FileUtils.rm_rf(record_dir.join("Backspin_record_naming"))
  end
end
