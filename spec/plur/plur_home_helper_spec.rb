require "spec_helper"

RSpec.describe PlurHomeHelper do
  include described_class

  describe "#without_plur_parallel_env" do
    it "keeps scrubbed vars out of Bundler.with_unbundled_env blocks" do
      # Bundler snapshots ORIGINAL_ENV at boot, so a var exported when the
      # suite started lives there too and with_unbundled_env restores it.
      ENV["NO_COLOR"] = "1"
      Bundler::ORIGINAL_ENV["NO_COLOR"] = "1"

      without_plur_parallel_env do
        expect(ENV["NO_COLOR"]).to be_nil
        Bundler.with_unbundled_env { expect(ENV["NO_COLOR"]).to be_nil }
      end

      expect(ENV["NO_COLOR"]).to eq("1")
      expect(Bundler::ORIGINAL_ENV["NO_COLOR"]).to eq("1")
    ensure
      ENV.delete("NO_COLOR")
      Bundler::ORIGINAL_ENV.delete("NO_COLOR")
    end
  end
end
