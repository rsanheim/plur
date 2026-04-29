require "spec_helper"

RSpec.describe PlurWatchHelper do
  include described_class

  describe "#watch_ready?" do
    it "does not report ready until every expected watcher directory is live" do
      ready_state = {count: 0, stable_since: nil}
      stderr = <<~ERR
        DEBUG - watch fullPath="s/self/live@/project/lib" event="create" type="watcher"
      ERR

      expect(watch_ready?(stderr, ready_state, ready_dirs: %w[lib spec])).to be(false)

      stderr << <<~ERR
        DEBUG - watch fullPath="s/self/live@/project/spec" event="create" type="watcher"
      ERR
      expect(watch_ready?(stderr, ready_state, ready_dirs: %w[lib spec])).to be(false)

      ready_state[:stable_since] = Time.now - PlurWatchHelper::READY_SETTLE_SECONDS - 0.1
      expect(watch_ready?(stderr, ready_state, ready_dirs: %w[lib spec])).to be(true)
    end

    it "preserves loose readiness when no expected directories are provided" do
      ready_state = {count: 0, stable_since: nil}
      stderr = <<~ERR
        DEBUG - watch fullPath="s/self/live@/project/lib" event="create" type="watcher"
      ERR

      expect(watch_ready?(stderr, ready_state)).to be(false)

      ready_state[:stable_since] = Time.now - PlurWatchHelper::READY_SETTLE_SECONDS - 0.1
      expect(watch_ready?(stderr, ready_state)).to be(true)
    end

    it "infers expected watcher directories from watch startup output" do
      ready_state = {count: 0, stable_since: nil}
      stderr = <<~ERR
        DEBUG - Watch directories after filtering dirs=[lib spec]
        DEBUG - watch fullPath="s/self/live@/project/lib" event="create" type="watcher"
      ERR

      expect(watch_ready?(stderr, ready_state, ready_dirs: :detected)).to be(false)

      stderr << <<~ERR
        DEBUG - watch fullPath="s/self/live@/project/spec" event="create" type="watcher"
      ERR
      expect(watch_ready?(stderr, ready_state, ready_dirs: :detected)).to be(false)

      ready_state[:stable_since] = Time.now - PlurWatchHelper::READY_SETTLE_SECONDS - 0.1
      expect(watch_ready?(stderr, ready_state, ready_dirs: :detected)).to be(true)
    end
  end
end
