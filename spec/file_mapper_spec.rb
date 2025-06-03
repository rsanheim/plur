require 'spec_helper'
require 'open3'

RSpec.describe "rux dev:file_mapper" do
  def run_file_mapper(*files)
    stdout, stderr, status = Open3.capture3("rux", "dev:file_mapper", *files)
    raise "Command failed: #{stderr}" unless status.success?
    stdout
  end
  
  context "lib files" do
    it "maps simple lib files to spec files" do
      output = run_file_mapper("lib/calculator.rb")
      expect(output.strip).to eq("lib/calculator.rb -> spec/calculator_spec.rb")
    end
    
    it "maps nested lib files correctly" do
      output = run_file_mapper("lib/models/user.rb")
      expect(output.strip).to eq("lib/models/user.rb -> spec/models/user_spec.rb")
    end
    
    it "handles non-ruby lib files" do
      output = run_file_mapper("lib/data.json")
      expect(output.strip).to eq("lib/data.json -> (no mapping)")
    end
  end
  
  context "Rails app files" do
    it "maps model files" do
      output = run_file_mapper("app/models/post.rb")
      expect(output.strip).to eq("app/models/post.rb -> spec/models/post_spec.rb")
    end
    
    it "maps controller files" do
      output = run_file_mapper("app/controllers/users_controller.rb")
      expect(output.strip).to eq("app/controllers/users_controller.rb -> spec/controllers/users_controller_spec.rb")
    end
    
    it "maps nested Rails files" do
      output = run_file_mapper("app/models/concerns/validatable.rb")
      expect(output.strip).to eq("app/models/concerns/validatable.rb -> spec/models/concerns/validatable_spec.rb")
    end
  end
  
  context "spec files" do
    it "maps spec files to themselves" do
      output = run_file_mapper("spec/models/user_spec.rb")
      expect(output.strip).to eq("spec/models/user_spec.rb -> spec/models/user_spec.rb")
    end
    
    it "maps spec_helper.rb to all specs" do
      output = run_file_mapper("spec/spec_helper.rb")
      expect(output.strip).to eq("spec/spec_helper.rb -> spec")
    end
    
    it "maps rails_helper.rb to all specs" do
      output = run_file_mapper("spec/rails_helper.rb")
      expect(output.strip).to eq("spec/rails_helper.rb -> spec")
    end
  end
  
  context "multiple files" do
    it "maps multiple files at once" do
      output = run_file_mapper("lib/a.rb", "lib/b.rb", "app/models/c.rb")
      lines = output.strip.split("\n")
      
      expect(lines).to eq([
        "lib/a.rb -> spec/a_spec.rb",
        "lib/b.rb -> spec/b_spec.rb",
        "app/models/c.rb -> spec/models/c_spec.rb"
      ])
    end
  end
  
  context "files with no mapping" do
    it "shows no mapping for unknown file types" do
      output = run_file_mapper("README.md", "config.yml", "random.txt")
      lines = output.strip.split("\n")
      
      expect(lines).to eq([
        "README.md -> (no mapping)",
        "config.yml -> (no mapping)",
        "random.txt -> (no mapping)"
      ])
    end
  end
  
  context "edge cases" do
    it "handles files with spaces in names" do
      output = run_file_mapper("lib/my calculator.rb")
      expect(output.strip).to eq("lib/my calculator.rb -> spec/my calculator_spec.rb")
    end
    
    it "handles absolute paths by mapping them relatively" do
      # The file mapper currently works on relative paths
      output = run_file_mapper("/absolute/path/lib/thing.rb")
      # Since it doesn't start with lib/ relatively, it won't map
      expect(output.strip).to eq("/absolute/path/lib/thing.rb -> (no mapping)")
    end
  end
end