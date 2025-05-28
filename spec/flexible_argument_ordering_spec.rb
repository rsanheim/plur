require 'spec_helper'
require 'open3'

RSpec.describe "Flexible argument ordering" do
  let(:rux_path) { File.expand_path('../rux/rux', __dir__) }

  context "with --no-color flag" do
    it "works with --no-color before file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} --no-color --dry-run spec/calculator_spec.rb")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 1 spec files")
        expect(output).to include("--no-color")
      end
    end

    it "works with --no-color after file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/calculator_spec.rb --no-color --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 1 spec files")
        expect(output).to include("--no-color")
      end
    end

    it "works with --no-color in the middle of file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/calculator_spec.rb --no-color spec/counter_spec.rb --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 2 spec files")
        expect(output).to include("--no-color")
      end
    end
  end

  context "with -n/--workers flag" do
    it "works with -n before file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} -n 4 --dry-run spec/calculator_spec.rb")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 1 spec files")
      end
    end

    it "works with -n after file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/calculator_spec.rb -n 4 --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 1 spec files")
      end
    end

    it "works with --workers after file arguments" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/calculator_spec.rb --workers 4 --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 1 spec files")
      end
    end
  end

  context "with directory arguments" do
    it "expands spec/ directory before flags" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/ --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 11 spec files")
      end
    end

    it "expands spec/ directory after flags" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} --dry-run spec/")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 11 spec files")
      end
    end

    it "expands spec/ directory with --no-color after" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/ --no-color --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 11 spec files")
        expect(output).to include("--no-color")
      end
    end
  end

  context "with multiple flags and arguments" do
    it "handles complex argument ordering" do
      Dir.chdir('rux-ruby') do
        output, status = Open3.capture2e("#{rux_path} spec/calculator_spec.rb --no-color -n 2 spec/counter_spec.rb --dry-run")
        
        expect(status.success?).to be true
        expect(output).to include("rux version")
        expect(output).to include("[dry-run] Found 2 spec files")
        expect(output).to include("--no-color")
        expect(output).to include("groups") # Should show grouping with 2 workers
      end
    end
  end
end