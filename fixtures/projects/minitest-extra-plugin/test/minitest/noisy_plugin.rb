# A minitest plugin discoverable on the -Itest load path. It announces itself
# during init so a run can prove whether it was activated. On minitest 6,
# where plugin loading is opt-in, a project that does not ask for this plugin
# must not have it activated - and plur must not activate it on the project's
# behalf.
module Minitest
  def self.plugin_noisy_init(options)
    options[:io].puts "NOISY_PLUGIN_LOADED"
  end
end
