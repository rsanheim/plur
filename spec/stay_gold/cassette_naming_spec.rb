require "stay_gold"

RSpec.describe "StayGold cassette naming" do
  let(:cassette_dir) { ROOT_PATH.join("tmp", "stay_gold") }
  
  describe "auto-generated names" do
    it "handles simple example names" do
      result = StayGold.record do
        Open3.capture3("echo test")
      end
      
      expect(result.cassette_path.to_s).to match(%r{StayGold_cassette_naming/auto-generated_names/handles_simple_example_names\.yaml$})
    end
    
    it "handles names with special characters" do
      result = StayGold.record do
        Open3.capture3("echo 'special chars!'")
      end
      
      expect(result.cassette_path.to_s).to match(%r{handles_names_with_special_characters\.yaml$})
    end
    
    context "nested contexts" do
      context "deeply nested" do
        it "includes all context levels" do
          result = StayGold.record do
            Open3.capture3("echo nested")
          end
          
          expect(result.cassette_path.to_s).to match(%r{StayGold_cassette_naming/auto-generated_names/nested_contexts/deeply_nested/includes_all_context_levels\.yaml$})
        end
      end
    end
  end
  
  after do
    # Clean up cassettes created during tests
    FileUtils.rm_rf(cassette_dir.join("StayGold_cassette_naming"))
  end
end